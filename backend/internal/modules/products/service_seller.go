package products

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/categories"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/inventory"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/stores"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/slug"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/validation"
)

type ImageInput struct {
	URL string
	Alt string
}

type VariantInput struct {
	Name  string
	SKU   string
	Price *decimal.Decimal
	Stock *int32
}

type CreateInput struct {
	Name        string
	Slug        string // optional: derived from Name
	Description string
	CategoryID  *uuid.UUID
	Images      []ImageInput
	Variants    []VariantInput
}

// UpdateInput: nil = unchanged. Images, when given, replace all images.
// Variants are changed through their own endpoints.
type UpdateInput struct {
	Name        *string
	Description *string
	CategoryID  *uuid.UUID
	Status      *string
	Images      *[]ImageInput
}

type VariantUpdateInput struct {
	Name  *string
	Price *decimal.Decimal
}

// storeOf resolves the caller's store. Every seller operation starts here, so
// the store always comes from the session, never from the request.
func (s *Service) storeOf(ctx context.Context, ownerID uuid.UUID) (stores.Store, error) {
	st, err := s.stores.GetByOwner(ctx, ownerID)
	if errors.Is(err, stores.ErrNotFound) {
		return stores.Store{}, ErrNoStore
	}
	return st, err
}

func (s *Service) ListMine(ctx context.Context, ownerID uuid.UUID) ([]SellerSummary, error) {
	st, err := s.storeOf(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	return s.repo.ListOwned(ctx, st.ID)
}

func (s *Service) GetMine(ctx context.Context, ownerID, productID uuid.UUID) (SellerProduct, error) {
	st, err := s.storeOf(ctx, ownerID)
	if err != nil {
		return SellerProduct{}, err
	}
	return s.loadOwned(ctx, st.ID, productID)
}

func (s *Service) loadOwned(ctx context.Context, storeID, productID uuid.UUID) (SellerProduct, error) {
	p, err := s.repo.GetOwned(ctx, storeID, productID)
	if err != nil {
		return SellerProduct{}, err
	}
	if p.Images, err = s.repo.Images(ctx, p.ID); err != nil {
		return SellerProduct{}, err
	}
	if p.Variants, err = s.repo.Variants(ctx, p.ID); err != nil {
		return SellerProduct{}, err
	}
	return p, nil
}

// Create adds a product with its variants, stock and images in ONE
// transaction. A product must never exist without a variant (it would have no
// price), and stock must never exist without its ledger row.
func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, in CreateInput) (SellerProduct, error) {
	st, err := s.storeOf(ctx, ownerID)
	if err != nil {
		return SellerProduct{}, err
	}

	errs := validation.Errors{}
	name := cleanText(errs, "name", in.Name, 3, 200)
	description := cleanText(errs, "description", in.Description, 0, 5000)
	sl := strings.TrimSpace(in.Slug)
	if sl == "" {
		sl = slug.Make(name)
	}
	if !slug.Valid(sl) {
		errs.Add("slug", "may only contain a-z, 0-9 and single hyphens")
	}
	if err := s.checkCategory(ctx, errs, in.CategoryID); err != nil {
		return SellerProduct{}, err
	}
	images := cleanImages(errs, in.Images)

	switch {
	case len(in.Variants) == 0:
		errs.Add("variants", "at least one variant is required")
	case len(in.Variants) > MaxVariants:
		errs.Add("variants", fmt.Sprintf("at most %d variants", MaxVariants))
	}
	type variant struct {
		name, sku string
		price     decimal.Decimal
		stock     int32
	}
	variants := make([]variant, 0, len(in.Variants))
	seenNames, seenSKUs := map[string]bool{}, map[string]bool{}
	for i, v := range in.Variants {
		field := fmt.Sprintf("variants[%d]", i)
		vn := cleanText(errs, field+".name", v.Name, 1, 100)
		sku := cleanSKU(errs, field+".sku", v.SKU)
		price := checkPrice(errs, field+".price", v.Price)
		stock := checkStock(errs, field+".stock", v.Stock)
		if seenNames[strings.ToLower(vn)] {
			errs.Add(field+".name", "is used by another variant")
		}
		if seenSKUs[sku] {
			errs.Add(field+".sku", "is used by another variant")
		}
		seenNames[strings.ToLower(vn)], seenSKUs[sku] = true, true
		variants = append(variants, variant{vn, sku, price, stock})
	}
	if err := errs.Err(); err != nil {
		return SellerProduct{}, err
	}

	var productID uuid.UUID
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)
		p, err := repo.Create(ctx, st.ID, *in.CategoryID, name, sl, description)
		if err != nil {
			return err
		}
		productID = p.ID
		if err := repo.ReplaceImages(ctx, p.ID, images); err != nil {
			return err
		}
		for i, v := range variants {
			created, err := repo.CreateVariant(ctx, p.ID, v.sku, v.name, v.price, int32(i))
			if err != nil {
				return err
			}
			if err := inventory.Create(ctx, tx, created.ID, v.stock, &ownerID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return SellerProduct{}, err
	}
	return s.loadOwned(ctx, st.ID, productID)
}

func (s *Service) Update(ctx context.Context, ownerID, productID uuid.UUID, in UpdateInput) (SellerProduct, error) {
	st, err := s.storeOf(ctx, ownerID)
	if err != nil {
		return SellerProduct{}, err
	}
	p, err := s.repo.GetOwned(ctx, st.ID, productID)
	if err != nil {
		return SellerProduct{}, err
	}

	errs := validation.Errors{}
	if in.Name != nil {
		p.Name = cleanText(errs, "name", *in.Name, 3, 200)
	}
	if in.Description != nil {
		p.Description = cleanText(errs, "description", *in.Description, 0, 5000)
	}
	if in.CategoryID != nil {
		if err := s.checkCategory(ctx, errs, in.CategoryID); err != nil {
			return SellerProduct{}, err
		}
		p.CategoryID = *in.CategoryID
	}
	if in.Status != nil {
		if *in.Status != StatusActive && *in.Status != StatusArchived {
			errs.Add("status", "must be active or archived")
		}
		p.Status = *in.Status
	}
	var images []Image
	if in.Images != nil {
		images = cleanImages(errs, *in.Images)
	}
	if err := errs.Err(); err != nil {
		return SellerProduct{}, err
	}

	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)
		if _, err := repo.UpdateOwned(ctx, st.ID, p); err != nil {
			return err
		}
		if in.Images != nil {
			return repo.ReplaceImages(ctx, p.ID, images)
		}
		return nil
	})
	if err != nil {
		return SellerProduct{}, err
	}
	return s.loadOwned(ctx, st.ID, productID)
}

// Archive hides a product from the public catalog. Products are never
// deleted: order history (later phases) keeps pointing at them.
func (s *Service) Archive(ctx context.Context, ownerID, productID uuid.UUID) error {
	status := StatusArchived
	_, err := s.Update(ctx, ownerID, productID, UpdateInput{Status: &status})
	return err
}

func (s *Service) AddVariant(ctx context.Context, ownerID, productID uuid.UUID, in VariantInput) (SellerProduct, error) {
	st, err := s.storeOf(ctx, ownerID)
	if err != nil {
		return SellerProduct{}, err
	}
	p, err := s.loadOwned(ctx, st.ID, productID)
	if err != nil {
		return SellerProduct{}, err
	}

	errs := validation.Errors{}
	name := cleanText(errs, "name", in.Name, 1, 100)
	sku := cleanSKU(errs, "sku", in.SKU)
	price := checkPrice(errs, "price", in.Price)
	stock := checkStock(errs, "stock", in.Stock)
	if len(p.Variants) >= MaxVariants {
		errs.Add("variants", fmt.Sprintf("at most %d variants", MaxVariants))
	}
	if err := errs.Err(); err != nil {
		return SellerProduct{}, err
	}

	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		repo := NewRepository(tx)
		order, err := repo.NextVariantSortOrder(ctx, productID)
		if err != nil {
			return err
		}
		v, err := repo.CreateVariant(ctx, productID, sku, name, price, order)
		if err != nil {
			return err
		}
		return inventory.Create(ctx, tx, v.ID, stock, &ownerID)
	})
	if err != nil {
		return SellerProduct{}, err
	}
	return s.loadOwned(ctx, st.ID, productID)
}

// UpdateVariant changes a variant's name or price. Past orders keep their own
// price snapshot (later phases), so a price change never rewrites history.
func (s *Service) UpdateVariant(ctx context.Context, ownerID, productID, variantID uuid.UUID, in VariantUpdateInput) (SellerProduct, error) {
	st, err := s.storeOf(ctx, ownerID)
	if err != nil {
		return SellerProduct{}, err
	}
	v, err := s.repo.GetOwnedVariant(ctx, st.ID, productID, variantID)
	if err != nil {
		return SellerProduct{}, err
	}

	errs := validation.Errors{}
	if in.Name != nil {
		v.Name = cleanText(errs, "name", *in.Name, 1, 100)
	}
	if in.Price != nil {
		v.Price = checkPrice(errs, "price", in.Price)
	}
	if err := errs.Err(); err != nil {
		return SellerProduct{}, err
	}
	if _, err := s.repo.UpdateVariant(ctx, v); err != nil {
		return SellerProduct{}, err
	}
	return s.loadOwned(ctx, st.ID, productID)
}

// SetStock sets a variant's available stock. The ownership check and the
// change run in the same transaction as the ledger entry.
func (s *Service) SetStock(ctx context.Context, ownerID, productID, variantID uuid.UUID, quantity *int32, reason string) (int32, error) {
	st, err := s.storeOf(ctx, ownerID)
	if err != nil {
		return 0, err
	}

	errs := validation.Errors{}
	qty := checkStock(errs, "quantity", quantity)
	reason = cleanText(errs, "reason", reason, 0, 200)
	if reason == "" {
		reason = defaultAdjustmentWhy
	}
	if err := errs.Err(); err != nil {
		return 0, err
	}

	var result int32
	err = database.WithTx(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := NewRepository(tx).GetOwnedVariant(ctx, st.ID, productID, variantID); err != nil {
			return err
		}
		var err error
		result, err = inventory.SetAvailable(ctx, tx, variantID, qty, &ownerID, reason)
		return err
	})
	return result, err
}

// ---- validation helpers -----------------------------------------------------

// checkCategory: the category must exist and be a leaf (a subcategory, or a
// top-level category without subcategories), so products never sit on a
// browse-only parent like "Elektronik" when "Audio" exists.
func (s *Service) checkCategory(ctx context.Context, errs validation.Errors, id *uuid.UUID) error {
	if id == nil {
		errs.Add("categoryId", "is required")
		return nil
	}
	if _, err := s.categories.GetByID(ctx, *id); errors.Is(err, categories.ErrNotFound) {
		errs.Add("categoryId", "does not exist")
		return nil
	} else if err != nil {
		return err
	}
	hasChildren, err := s.categories.HasChildren(ctx, *id)
	if err != nil {
		return err
	}
	if hasChildren {
		errs.Add("categoryId", "choose a subcategory")
	}
	return nil
}

func cleanText(errs validation.Errors, field, v string, minLen, maxLen int) string {
	v = strings.TrimSpace(v)
	switch n := utf8.RuneCountInString(v); {
	case n == 0 && minLen > 0:
		errs.Add(field, "is required")
	case n < minLen:
		errs.Add(field, fmt.Sprintf("must be at least %d characters", minLen))
	case n > maxLen:
		errs.Add(field, fmt.Sprintf("must be at most %d characters", maxLen))
	}
	return v
}

var skuPattern = regexp.MustCompile(`^[A-Z0-9][A-Z0-9._-]{0,63}$`)

// cleanSKU uppercases so "kmj-01" and "KMJ-01" cannot both exist.
func cleanSKU(errs validation.Errors, field, sku string) string {
	sku = strings.ToUpper(strings.TrimSpace(sku))
	switch {
	case sku == "":
		errs.Add(field, "is required")
	case !skuPattern.MatchString(sku):
		errs.Add(field, "may only contain A-Z, 0-9, '.', '_' and '-' (max 64)")
	}
	return sku
}

func checkPrice(errs validation.Errors, field string, p *decimal.Decimal) decimal.Decimal {
	switch {
	case p == nil:
		errs.Add(field, "is required")
		return decimal.Zero
	case p.IsNegative():
		errs.Add(field, "cannot be negative")
	case p.GreaterThan(maxPrice):
		errs.Add(field, "is too large")
	case !p.Equal(p.Round(2)):
		errs.Add(field, "may have at most 2 decimal places")
	}
	return *p
}

func checkStock(errs validation.Errors, field string, q *int32) int32 {
	switch {
	case q == nil:
		errs.Add(field, "is required")
		return 0
	case *q < 0:
		errs.Add(field, "cannot be negative")
	case *q > inventory.MaxQuantity:
		errs.Add(field, fmt.Sprintf("must be at most %d", inventory.MaxQuantity))
	}
	return *q
}

func cleanImages(errs validation.Errors, in []ImageInput) []Image {
	if len(in) > MaxImages {
		errs.Add("images", fmt.Sprintf("at most %d images", MaxImages))
		return nil
	}
	out := make([]Image, 0, len(in))
	for i, img := range in {
		field := fmt.Sprintf("images[%d].url", i)
		raw := strings.TrimSpace(img.URL)
		u, err := url.Parse(raw)
		// Only absolute http(s) URLs: never javascript: or data: in an <img src>.
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(raw) > 2000 {
			errs.Add(field, "must be an http(s) URL")
			continue
		}
		alt := cleanText(errs, fmt.Sprintf("images[%d].alt", i), img.Alt, 0, 200)
		out = append(out, Image{URL: raw, Alt: alt})
	}
	return out
}

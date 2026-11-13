// Command seed-dev fills an EMPTY development database with demo data:
// a two-level category tree, four seller accounts with a store each, and
// their products (the data the UI prototype used to mock).
//
//	make seed-dev
//
// It goes through the same services as the API, so demo data obeys every
// validation rule, uses the same transactions and writes inventory ledger
// rows like real data would. It refuses to run in production and does
// nothing if categories already exist.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/config"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/database"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/auth"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/categories"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/products"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/stores"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/password"
)

// SellerPassword is for local demo accounts only.
const SellerPassword = "penjual-123"

type categorySeed struct {
	name     string
	children []string
}

var categoryTree = []categorySeed{
	{"Elektronik", []string{"Audio", "Aksesoris Komputer", "Gadget & Wearable", "Charger & Kabel"}},
	{"Fashion Pria", []string{"Atasan Pria", "Busana Muslim Pria"}},
	{"Fashion Wanita", []string{"Atasan Wanita", "Busana Muslim Wanita"}},
	{"Rumah Tangga", []string{"Dapur", "Penyimpanan", "Dekorasi & Lampu", "Kamar Tidur"}},
	{"Olahraga", []string{"Lari", "Fitness", "Raket"}},
	{"Buku & Hobi", []string{"Buku", "Puzzle & Mainan"}},
}

type variantSeed struct {
	name  string
	price string
	stock int32
}

type productSeed struct {
	name, category, description string // category = subcategory slug
	variants                    []variantSeed
	sold                        int32
	archived                    bool
}

type storeSeed struct {
	email, owner, name, code, location, description string
	products                                        []productSeed
}

var storeSeeds = []storeSeed{
	{"penjual1@contoh.id", "Budi Santoso", "Gadget Nusantara", "GN", "Jakarta Barat",
		"Aksesoris dan perangkat elektronik harian dengan garansi toko 7 hari.", []productSeed{
			{"Earbuds Bluetooth Pro ANC", "audio", "Earbuds nirkabel dengan peredam bising aktif, baterai hingga 30 jam bersama case, dan mikrofon jernih untuk panggilan.",
				[]variantSeed{{"Hitam", "349000", 40}, {"Putih", "349000", 25}}, 3200, false},
			{"Power Bank 20.000 mAh Fast Charging", "charger-kabel", "Kapasitas besar dengan dua port USB dan satu USB-C 22,5W. Cocok untuk perjalanan jauh.",
				[]variantSeed{{"20.000 mAh", "259000", 120}}, 5100, false},
			{"Smartwatch Fit 2 Layar AMOLED", "gadget-wearable", "Layar AMOLED 1,43 inci, pemantau detak jantung, tahan air 5 ATM, dan 100+ mode olahraga.",
				[]variantSeed{{"Hitam", "599000", 15}, {"Silver", "629000", 0}}, 980, false},
			{"Keyboard Mekanikal 75% Hot-swap", "aksesoris-komputer", "Layout 75% ringkas, switch dapat diganti tanpa solder, tersedia koneksi kabel dan Bluetooth.",
				[]variantSeed{{"Brown Switch", "499000", 22}, {"Blue Switch", "489000", 10}}, 740, false},
			{"Mouse Wireless Silent Click", "aksesoris-komputer", "Klik senyap, sensor 1600 DPI, dan baterai tahan hingga 12 bulan.",
				[]variantSeed{{"Hitam", "129000", 200}}, 8900, false},
			{"Kabel USB-C Braided", "charger-kabel", "Kabel anyaman nilon anti kusut, mendukung pengisian cepat 60W dan transfer data.",
				[]variantSeed{{"1 meter", "39000", 300}, {"2 meter", "59000", 180}}, 21000, false},
		}},
	{"penjual2@contoh.id", "Siti Rahayu", "Batik Kita", "BK", "Surabaya",
		"Batik dan busana tradisional modern langsung dari pengrajin.", []productSeed{
			{"Kemeja Batik Pria Motif Parang", "atasan-pria", "Batik cap katun halus dengan motif parang klasik. Nyaman dipakai untuk acara formal maupun santai.",
				[]variantSeed{{"M", "189000", 18}, {"L", "189000", 24}, {"XL", "199000", 9}}, 2300, false},
			{"Kaos Polos Cotton Combed 24s", "atasan-pria", "Bahan cotton combed 24s yang adem dan tidak mudah melar. Tersedia banyak warna dasar.",
				[]variantSeed{{"M", "59000", 80}, {"L", "59000", 95}, {"XL", "65000", 40}}, 17500, false},
			{"Gamis Batik Modern Rayon", "busana-muslim-wanita", "Gamis rayon adem dengan kombinasi batik modern, dilengkapi saku dan resleting depan busui friendly.",
				[]variantSeed{{"S", "259000", 12}, {"M", "259000", 20}, {"L", "259000", 14}}, 1500, false},
			{"Tunik Bordir Elegan", "atasan-wanita", "Tunik bordir halus di bagian dada dan lengan, jatuh nyaman dan mudah dipadukan.",
				[]variantSeed{{"All size", "175000", 33}}, 640, false},
			{"Sarung Tenun Premium", "busana-muslim-pria", "Sarung tenun halus berbahan katun, warna tidak luntur, cocok untuk ibadah dan hadiah.",
				[]variantSeed{{"Motif Kotak", "145000", 60}}, 410, false},
			{"Hijab Segi Empat Voal Premium", "busana-muslim-wanita", "Voal ultrafine yang ringan dan mudah dibentuk, jahit tepi rapi dan tidak menerawang.",
				[]variantSeed{{"Dusty Pink", "49000", 150}, {"Sage", "49000", 120}, {"Hitam", "49000", 300}}, 12300, false},
			{"Kemeja Batik Edisi Lama", "atasan-pria", "Produk yang sudah diarsipkan penjual; tidak boleh muncul di katalog publik.",
				[]variantSeed{{"M", "99000", 5}}, 30, true},
		}},
	{"penjual3@contoh.id", "Agus Wijaya", "Rumah Asri", "RA", "Bandung",
		"Perlengkapan rumah, dapur, dan dekorasi untuk hunian yang nyaman.", []productSeed{
			{"Set Panci Anti Lengket 5 Pcs", "dapur", "Lapisan anti lengket bebas PFOA, cocok untuk kompor gas maupun induksi.",
				[]variantSeed{{"5 Pcs", "459000", 30}}, 1900, false},
			{"Rak Sepatu Susun 4 Tingkat", "penyimpanan", "Rangka besi kokoh dengan rak tertutup. Muat hingga 12 pasang sepatu.",
				[]variantSeed{{"Putih", "189000", 45}}, 3400, false},
			{"Lampu Meja LED Dimmable", "dekorasi-lampu", "3 mode warna cahaya, 5 tingkat kecerahan, dan port USB untuk mengisi daya ponsel.",
				[]variantSeed{{"Putih", "135000", 70}}, 1250, false},
			{"Diffuser Aromaterapi Ultrasonik", "dekorasi-lampu", "Tangki 300 ml dengan lampu malam, mati otomatis saat air habis.",
				[]variantSeed{{"Kayu", "175000", 0}}, 870, false},
			{"Sprei Katun Queen Size", "kamar-tidur", "Katun halus 160x200 cm dengan tinggi kasur hingga 30 cm, termasuk dua sarung bantal.",
				[]variantSeed{{"Motif Daun", "249000", 25}, {"Motif Garis", "249000", 31}}, 4200, false},
			{"Botol Minum Stainless 750ml", "dapur", "Menahan dingin hingga 24 jam dan panas hingga 12 jam. Tutup anti bocor.",
				[]variantSeed{{"Hijau", "89000", 140}, {"Hitam", "89000", 180}}, 14000, false},
			{"Puzzle 1000 Keping Pemandangan", "puzzle-mainan", "Puzzle karton tebal dengan hasil cetak tajam, ukuran jadi 70x50 cm.",
				[]variantSeed{{"1000 keping", "119000", 28}}, 520, false},
		}},
	{"penjual4@contoh.id", "Dewi Lestari", "Sportivo Store", "SP", "Yogyakarta",
		"Peralatan olahraga dan buku hobi untuk gaya hidup aktif.", []productSeed{
			{"Sepatu Lari Ultralight", "lari", "Bobot hanya 230 gram dengan midsole responsif dan upper rajut yang bernapas.",
				[]variantSeed{{"40", "549000", 6}, {"41", "549000", 11}, {"42", "549000", 14}, {"43", "549000", 5}}, 2100, false},
			{"Matras Yoga TPE 8mm", "fitness", "Permukaan anti selip dua sisi, ramah lingkungan, dilengkapi tali pengikat.",
				[]variantSeed{{"Ungu", "149000", 55}, {"Biru", "149000", 48}}, 5300, false},
			{"Dumbbell Adjustable 2x10kg", "fitness", "Beban dapat diatur dari 2,5 sampai 10 kg per sisi, pegangan karet anti selip.",
				[]variantSeed{{"2 x 10 kg", "699000", 8}}, 520, false},
			{"Raket Badminton Karbon", "raket", "Frame karbon ringan 85 gram dengan keseimbangan head-heavy untuk smash tajam.",
				[]variantSeed{{"Hitam-Merah", "329000", 19}}, 1100, false},
			{"Buku Latihan Konsisten 30 Hari", "buku", "Panduan latihan harian dengan lembar pelacak kebiasaan dan catatan progres.",
				[]variantSeed{{"Paperback", "79000", 90}}, 360, false},
		}},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "seed-dev:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.IsProduction() {
		return errors.New("refusing to load demo data with APP_ENV=production")
	}
	slog.SetLogLoggerLevel(slog.LevelWarn)

	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.DB)
	if err != nil {
		return err
	}
	defer pool.Close()

	catSvc := categories.NewService(pool)
	existing, err := catSvc.List(ctx)
	if err != nil {
		return err
	}
	if len(existing) > 0 {
		fmt.Println("categories already exist; database looks seeded, nothing to do")
		return nil
	}

	authSvc, err := auth.NewService(pool, auth.Config{
		SessionTTL: cfg.SessionTTL,
		Password:   password.Params{MemoryKiB: cfg.Argon2.MemoryKiB, Iterations: cfg.Argon2.Iterations, Parallelism: cfg.Argon2.Parallelism},
	})
	if err != nil {
		return err
	}
	storeSvc := stores.NewService(pool)
	productSvc := products.NewService(pool)
	productRepo := products.NewRepository(pool)

	categoryIDs := map[string]uuid.UUID{} // subcategory slug -> id
	for i, top := range categoryTree {
		parent, err := catSvc.Create(ctx, categories.Input{Name: top.name, SortOrder: int32(i)})
		if err != nil {
			return fmt.Errorf("category %q: %w", top.name, err)
		}
		for j, child := range top.children {
			c, err := catSvc.Create(ctx, categories.Input{Name: child, ParentID: &parent.ID, SortOrder: int32(j)})
			if err != nil {
				return fmt.Errorf("category %q: %w", child, err)
			}
			categoryIDs[c.Slug] = c.ID
		}
	}

	productCount := 0
	for _, ss := range storeSeeds {
		owner, err := authSvc.CreateUserWithRole(ctx, auth.RegisterInput{Name: ss.owner, Email: ss.email, Password: SellerPassword}, authctx.RoleCustomer)
		if err != nil {
			return fmt.Errorf("seller %s: %w", ss.email, err)
		}
		if _, err := storeSvc.Open(ctx, owner.ID, stores.OpenInput{Name: ss.name, Description: ss.description, Location: ss.location}); err != nil {
			return fmt.Errorf("store %q: %w", ss.name, err)
		}

		for pi, ps := range ss.products {
			catID, ok := categoryIDs[ps.category]
			if !ok {
				return fmt.Errorf("product %q: unknown category %q", ps.name, ps.category)
			}
			variants := make([]products.VariantInput, len(ps.variants))
			for vi, vs := range ps.variants {
				price := decimal.RequireFromString(vs.price)
				stock := vs.stock
				variants[vi] = products.VariantInput{
					Name:  vs.name,
					SKU:   fmt.Sprintf("%s-%02d-%02d", ss.code, pi+1, vi+1),
					Price: &price,
					Stock: &stock,
				}
			}
			p, err := productSvc.Create(ctx, owner.ID, products.CreateInput{
				Name: ps.name, Description: ps.description, CategoryID: &catID, Variants: variants,
			})
			if err != nil {
				return fmt.Errorf("product %q: %w", ps.name, err)
			}
			if err := productRepo.SetSoldCount(ctx, p.ID, ps.sold); err != nil {
				return err
			}
			if ps.archived {
				if err := productSvc.Archive(ctx, owner.ID, p.ID); err != nil {
					return err
				}
			}
			productCount++
		}
	}

	fmt.Printf("seeded %d categories, %d stores, %d products\n", len(categoryIDs)+len(categoryTree), len(storeSeeds), productCount)
	fmt.Printf("seller logins: penjual1..4@contoh.id / %s (local demo only)\n", SellerPassword)
	return nil
}

// Package router builds the Gin engine and registers every route.
// Reading this one file shows the whole public surface of the API.
package router

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/config"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/middleware"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/auth"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/cart"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/categories"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/health"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/products"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/stores"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/modules/users"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/authctx"
	"github.com/RavaGhiffari/ecommerceCopyGoReact/backend/internal/platform/response"
)

type Handlers struct {
	Health     *health.Handler
	Auth       *auth.Handler
	Users      *users.Handler
	Categories *categories.Handler
	Stores     *stores.Handler
	Products   *products.Handler
	Cart       *cart.Handler
}

func New(cfg config.Config, h Handlers, resolveSession middleware.SessionResolver) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	// We are not behind a reverse proxy yet, so never trust X-Forwarded-For.
	_ = r.SetTrustedProxies(nil)

	// Middleware runs top to bottom on the way in, bottom to top on the way out:
	//   RequestID -> Logging -> Recovery -> CORS -> Timeout -> SameOrigin -> Authenticate -> handler
	// Logging sits outside Recovery so a recovered panic is still logged as a 500.
	// SameOrigin runs before Authenticate: a forged cross-site request is
	// rejected before we even look at its cookie.
	r.Use(
		middleware.RequestID(),
		middleware.Logging(),
		middleware.Recovery(),
		middleware.CORS(cfg.CORSAllowedOrigins),
		middleware.Timeout(cfg.RequestTimeout),
		middleware.SameOrigin(cfg.CORSAllowedOrigins),
		middleware.Authenticate(resolveSession),
	)

	// Unknown routes still answer in the standard envelope.
	r.HandleMethodNotAllowed = true
	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, response.CodeNotFound, "Route not found", nil)
	})
	r.NoMethod(func(c *gin.Context) {
		response.Fail(c, http.StatusMethodNotAllowed, response.CodeMethodNotAllowed, "Method not allowed", nil)
	})

	// Health endpoints sit outside /api/v1: they are for infrastructure, not clients.
	r.GET("/health", h.Health.Live)
	r.GET("/ready", h.Health.Ready)

	api := r.Group("/api/v1")

	// ---- Public: no login needed ---------------------------------------------
	authRoutes := api.Group("/auth")
	authRoutes.POST("/register", h.Auth.Register)
	authRoutes.POST("/login", h.Auth.Login)
	authRoutes.POST("/logout", h.Auth.Logout)

	api.GET("/categories", h.Categories.List)
	api.GET("/products", h.Products.List)
	api.GET("/products/:id/related", h.Products.Related)
	api.GET("/stores/:slug", h.Stores.Public)
	api.GET("/stores/:slug/products/:productSlug", h.Products.Detail)

	// ---- Any logged-in user ---------------------------------------------------
	signedIn := api.Group("", middleware.RequireAuth())
	signedIn.GET("/me", h.Users.GetMe)
	signedIn.PATCH("/me", h.Users.UpdateMe)
	// The cart always belongs to the session user (no IDs from the client).
	signedIn.GET("/cart", h.Cart.Get)
	signedIn.POST("/cart/items", h.Cart.AddItem)
	signedIn.PATCH("/cart/items/:id", h.Cart.UpdateItem)
	signedIn.DELETE("/cart/items/:id", h.Cart.RemoveItem)
	// Opening a store is how a customer becomes a seller.
	signedIn.POST("/seller/store", h.Stores.Open)

	// ---- Sellers: everything acts on the caller's own store -------------------
	seller := api.Group("/seller", middleware.RequireRole(authctx.RoleSeller))
	seller.GET("/store", h.Stores.Mine)
	seller.PATCH("/store", h.Stores.UpdateMine)
	seller.GET("/products", h.Products.ListMine)
	seller.POST("/products", h.Products.Create)
	seller.GET("/products/:id", h.Products.GetMine)
	seller.PATCH("/products/:id", h.Products.Update)
	seller.DELETE("/products/:id", h.Products.Archive)
	seller.POST("/products/:id/variants", h.Products.AddVariant)
	seller.PATCH("/products/:id/variants/:variantId", h.Products.UpdateVariant)
	seller.PUT("/products/:id/variants/:variantId/stock", h.Products.SetStock)

	// ---- Admins ----------------------------------------------------------------
	admin := api.Group("/admin", middleware.RequireRole(authctx.RoleAdmin))
	admin.POST("/categories", h.Categories.Create)
	admin.PUT("/categories/:id", h.Categories.Replace)

	return r
}

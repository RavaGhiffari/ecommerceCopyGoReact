import { createBrowserRouter } from 'react-router-dom'
import { AdminLayout } from '../features/admin/AdminLayout'
import { RequireAdmin } from '../features/admin/RequireAdmin'
import { RequireAuth } from '../features/auth/RequireAuth'
import { StatusPage } from '../features/status/StatusPage'
import { MainLayout } from '../layouts/MainLayout'
import { AccountPage } from '../pages/AccountPage'
import { AdminAuditLogsPage } from '../pages/AdminAuditLogsPage'
import { AdminCategoriesPage } from '../pages/AdminCategoriesPage'
import { AdminDashboardPage } from '../pages/AdminDashboardPage'
import { AdminOrderDetailPage } from '../pages/AdminOrderDetailPage'
import { AdminOrdersPage } from '../pages/AdminOrdersPage'
import { AdminReviewsPage } from '../pages/AdminReviewsPage'
import { AdminStoresPage } from '../pages/AdminStoresPage'
import { AdminUsersPage } from '../pages/AdminUsersPage'
import { CartPage } from '../pages/CartPage'
import { CheckoutPage } from '../pages/CheckoutPage'
import { HomePage } from '../pages/HomePage'
import { LoginPage } from '../pages/LoginPage'
import { NotFoundPage } from '../pages/NotFoundPage'
import { OrderDetailPage } from '../pages/OrderDetailPage'
import { OrdersPage } from '../pages/OrdersPage'
import { PaymentPage } from '../pages/PaymentPage'
import { ProductDetailPage } from '../pages/ProductDetailPage'
import { ProductListPage } from '../pages/ProductListPage'
import { RegisterPage } from '../pages/RegisterPage'
import { RequireSeller } from '../features/seller/RequireSeller'
import { SellerEditProductPage } from '../pages/SellerEditProductPage'
import { SellerNewProductPage } from '../pages/SellerNewProductPage'
import { SellerOrderDetailPage } from '../pages/SellerOrderDetailPage'
import { SellerOrdersPage } from '../pages/SellerOrdersPage'
import { SellerPage } from '../pages/SellerPage'
import { StorePage } from '../pages/StorePage'
import { WishlistPage } from '../pages/WishlistPage'

export const router = createBrowserRouter([
  {
    element: <MainLayout />,
    children: [
      { index: true, element: <HomePage /> },
      { path: 'products', element: <ProductListPage /> },
      { path: 'categories/:categorySlug', element: <ProductListPage /> },
      { path: 'stores/:storeSlug', element: <StorePage /> },
      // Store-scoped product URL: slugs are unique per store (UNIQUE (store_id, slug)).
      { path: 'stores/:storeSlug/products/:productSlug', element: <ProductDetailPage /> },

      { path: 'login', element: <LoginPage /> },
      { path: 'register', element: <RegisterPage /> },
      // RequireAuth is UX only; the API enforces access on its own.
      { path: 'account', element: <RequireAuth><AccountPage /></RequireAuth> },
      { path: 'cart', element: <RequireAuth><CartPage /></RequireAuth> },
      { path: 'checkout', element: <RequireAuth><CheckoutPage /></RequireAuth> },
      { path: 'checkout/:id', element: <RequireAuth><PaymentPage /></RequireAuth> },
      { path: 'orders', element: <RequireAuth><OrdersPage /></RequireAuth> },
      { path: 'orders/:id', element: <RequireAuth><OrderDetailPage /></RequireAuth> },
      { path: 'wishlist', element: <RequireAuth><WishlistPage /></RequireAuth> },
      { path: 'seller', element: <RequireAuth><SellerPage /></RequireAuth> },
      { path: 'seller/products/new', element: <RequireSeller><SellerNewProductPage /></RequireSeller> },
      { path: 'seller/products/:id', element: <RequireSeller><SellerEditProductPage /></RequireSeller> },
      { path: 'seller/orders', element: <RequireSeller><SellerOrdersPage /></RequireSeller> },
      { path: 'seller/orders/:id', element: <RequireSeller><SellerOrderDetailPage /></RequireSeller> },

      // Admin panel (6e). RequireAdmin is UX only; every /api/v1/admin route checks the role.
      {
        path: 'admin',
        element: <RequireAdmin><AdminLayout /></RequireAdmin>,
        children: [
          { index: true, element: <AdminDashboardPage /> },
          { path: 'users', element: <AdminUsersPage /> },
          { path: 'stores', element: <AdminStoresPage /> },
          { path: 'categories', element: <AdminCategoriesPage /> },
          { path: 'orders', element: <AdminOrdersPage /> },
          { path: 'orders/:id', element: <AdminOrderDetailPage /> },
          { path: 'reviews', element: <AdminReviewsPage /> },
          { path: 'audit-logs', element: <AdminAuditLogsPage /> },
        ],
      },

      { path: 'status', element: <StatusPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])

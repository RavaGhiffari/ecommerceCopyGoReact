import type { ComponentType, ReactNode } from 'react'
import { createBrowserRouter, Outlet, type RouteObject } from 'react-router-dom'
import { FirstLoad } from './FirstLoad'
import { RequireAdmin } from '../features/admin/RequireAdmin'
import { RequireAuth } from '../features/auth/RequireAuth'
import { RequireSeller } from '../features/seller/RequireSeller'
import { StatusPage } from '../features/status/StatusPage'
import { MainLayout } from '../layouts/MainLayout'
// The storefront (browse, product, sign in) is in the first bundle. Everything
// behind a login (cart, checkout, orders, seller, admin) is loaded on first
// visit, so a shopper who only browses never downloads the admin panel.
import { HomePage } from '../pages/HomePage'
import { LoginPage } from '../pages/LoginPage'
import { NotFoundPage } from '../pages/NotFoundPage'
import { ProductDetailPage } from '../pages/ProductDetailPage'
import { ProductListPage } from '../pages/ProductListPage'
import { RegisterPage } from '../pages/RegisterPage'
import { StorePage } from '../pages/StorePage'

type Guard = (props: { children: ReactNode }) => ReactNode

// A route whose page module is imported on first navigation. React Router
// shows the previous page until the chunk has arrived (no Suspense needed).
// `guard` wraps the page exactly as the old eager `<RequireX><Page /></RequireX>`.
function page<M, K extends keyof M>(load: () => Promise<M>, name: K, guard?: Guard): Pick<RouteObject, 'lazy'> {
  return {
    lazy: async () => {
      const Page = (await load())[name] as ComponentType
      return { element: guard ? guard({ children: <Page /> }) : <Page /> }
    },
  }
}

const auth: Guard = (p) => <RequireAuth>{p.children}</RequireAuth>
const seller: Guard = (p) => <RequireSeller>{p.children}</RequireSeller>

export const router = createBrowserRouter([
  {
    element: <MainLayout />,
    HydrateFallback: FirstLoad,
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
      { path: 'account', ...page(() => import('../pages/AccountPage'), 'AccountPage', auth) },
      { path: 'cart', ...page(() => import('../pages/CartPage'), 'CartPage', auth) },
      { path: 'checkout', ...page(() => import('../pages/CheckoutPage'), 'CheckoutPage', auth) },
      { path: 'checkout/:id', ...page(() => import('../pages/PaymentPage'), 'PaymentPage', auth) },
      { path: 'orders', ...page(() => import('../pages/OrdersPage'), 'OrdersPage', auth) },
      { path: 'orders/:id', ...page(() => import('../pages/OrderDetailPage'), 'OrderDetailPage', auth) },
      { path: 'wishlist', ...page(() => import('../pages/WishlistPage'), 'WishlistPage', auth) },
      { path: 'seller', ...page(() => import('../pages/SellerPage'), 'SellerPage', auth) },
      { path: 'seller/products/new', ...page(() => import('../pages/SellerNewProductPage'), 'SellerNewProductPage', seller) },
      { path: 'seller/products/:id', ...page(() => import('../pages/SellerEditProductPage'), 'SellerEditProductPage', seller) },
      { path: 'seller/orders', ...page(() => import('../pages/SellerOrdersPage'), 'SellerOrdersPage', seller) },
      { path: 'seller/orders/:id', ...page(() => import('../pages/SellerOrderDetailPage'), 'SellerOrderDetailPage', seller) },

      // Admin panel (6e). RequireAdmin is UX only; every /api/v1/admin route checks the role.
      // The guard is eager (it is tiny); the layout and every admin page load on demand.
      {
        path: 'admin',
        element: <RequireAdmin><Outlet /></RequireAdmin>,
        children: [
          {
            ...page(() => import('../features/admin/AdminLayout'), 'AdminLayout'),
            children: [
              { index: true, ...page(() => import('../pages/AdminDashboardPage'), 'AdminDashboardPage') },
              { path: 'users', ...page(() => import('../pages/AdminUsersPage'), 'AdminUsersPage') },
              { path: 'stores', ...page(() => import('../pages/AdminStoresPage'), 'AdminStoresPage') },
              { path: 'categories', ...page(() => import('../pages/AdminCategoriesPage'), 'AdminCategoriesPage') },
              { path: 'orders', ...page(() => import('../pages/AdminOrdersPage'), 'AdminOrdersPage') },
              { path: 'orders/:id', ...page(() => import('../pages/AdminOrderDetailPage'), 'AdminOrderDetailPage') },
              { path: 'reviews', ...page(() => import('../pages/AdminReviewsPage'), 'AdminReviewsPage') },
              { path: 'audit-logs', ...page(() => import('../pages/AdminAuditLogsPage'), 'AdminAuditLogsPage') },
            ],
          },
        ],
      },

      { path: 'status', element: <StatusPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])

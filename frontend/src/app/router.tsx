import { createBrowserRouter } from 'react-router-dom'
import { RequireAuth } from '../features/auth/RequireAuth'
import { StatusPage } from '../features/status/StatusPage'
import { MainLayout } from '../layouts/MainLayout'
import { AccountPage } from '../pages/AccountPage'
import { CartPage } from '../pages/CartPage'
import { HomePage } from '../pages/HomePage'
import { LoginPage } from '../pages/LoginPage'
import { NotFoundPage } from '../pages/NotFoundPage'
import { ProductDetailPage } from '../pages/ProductDetailPage'
import { ProductListPage } from '../pages/ProductListPage'
import { RegisterPage } from '../pages/RegisterPage'
import { RequireSeller } from '../features/seller/RequireSeller'
import { SellerEditProductPage } from '../pages/SellerEditProductPage'
import { SellerNewProductPage } from '../pages/SellerNewProductPage'
import { SellerPage } from '../pages/SellerPage'
import { StorePage } from '../pages/StorePage'

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
      { path: 'seller', element: <RequireAuth><SellerPage /></RequireAuth> },
      { path: 'seller/products/new', element: <RequireSeller><SellerNewProductPage /></RequireSeller> },
      { path: 'seller/products/:id', element: <RequireSeller><SellerEditProductPage /></RequireSeller> },

      { path: 'status', element: <StatusPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])

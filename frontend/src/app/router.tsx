import { createBrowserRouter } from 'react-router-dom'
import { StatusPage } from '../features/status/StatusPage'
import { MainLayout } from '../layouts/MainLayout'
import { HomePage } from '../pages/HomePage'
import { NotFoundPage } from '../pages/NotFoundPage'
import { ProductDetailPage } from '../pages/ProductDetailPage'
import { ProductListPage } from '../pages/ProductListPage'
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
      { path: 'status', element: <StatusPage /> },
      { path: '*', element: <NotFoundPage /> },
    ],
  },
])

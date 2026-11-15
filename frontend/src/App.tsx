import { RouterProvider } from 'react-router-dom'
import { Toaster } from '@/components/ui/sonner'
import { router } from './app/router'
import { AuthProvider } from './features/auth/AuthProvider'
import { CartProvider } from './features/cart/CartProvider'

export function App() {
  return (
    <AuthProvider>
      <CartProvider>
        <RouterProvider router={router} />
        <Toaster position="top-center" richColors closeButton />
      </CartProvider>
    </AuthProvider>
  )
}

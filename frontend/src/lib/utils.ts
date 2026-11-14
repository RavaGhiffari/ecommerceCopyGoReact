import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

/**
 * Joins class names and resolves Tailwind conflicts: the last one wins, so
 * `cn('px-2', 'px-4')` is `'px-4'`. Lets callers override a shadcn
 * component's default classes through `className`.
 */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

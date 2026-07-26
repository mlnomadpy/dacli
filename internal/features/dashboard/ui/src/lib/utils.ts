import { type ClassValue, clsx } from 'clsx'
import { twMerge } from 'tailwind-merge'

// The shadcn-vue class-name helper: merge conditional clsx() classes and let
// tailwind-merge win the last conflicting Tailwind utility. Every `ui/`
// component composes its class list through this.
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

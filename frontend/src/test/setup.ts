import '@testing-library/jest-dom/vitest'
import { cleanup } from '@testing-library/react'
import { afterEach, vi } from 'vitest'

Object.defineProperty(window, 'scrollTo', { value: vi.fn(), writable: true })
const originalMatchMedia = window.matchMedia

afterEach(() => {
  cleanup()
  vi.clearAllMocks()
  vi.unstubAllGlobals()
  if (originalMatchMedia) Object.defineProperty(window, 'matchMedia', { configurable: true, writable: true, value: originalMatchMedia })
  else Reflect.deleteProperty(window, 'matchMedia')
})

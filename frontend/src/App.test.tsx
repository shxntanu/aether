// @vitest-environment jsdom

import '@testing-library/jest-dom/vitest'
import { render, screen } from '@testing-library/react'
import { afterEach, expect, test, vi } from 'vitest'

import App from './App'

afterEach(() => {
  vi.unstubAllGlobals()
})

test('presents the coming-soon message without requesting API status', () => {
  const fetch = vi.fn()
  vi.stubGlobal('fetch', fetch)
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockReturnValue({
      addEventListener: vi.fn(),
      matches: true,
      removeEventListener: vi.fn(),
    }),
  )

  render(<App />)

  expect(screen.getByRole('heading', { name: 'AETHER' })).toBeVisible()
  expect(screen.getByText('COMING SOON')).toBeVisible()
  expect(fetch).not.toHaveBeenCalled()
})

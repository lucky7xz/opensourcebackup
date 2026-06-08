import { defineConfig } from 'vitest/config'

// Unit tests run in a plain Node environment — the modules under test are pure
// (no DOM). Component/DOM tests can opt into 'jsdom' per-file later if needed.
export default defineConfig({
  test: {
    environment: 'node',
    include: ['src/**/*.test.ts'],
  },
})

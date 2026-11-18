import { defineConfig, mergeConfig } from 'vitest/config'
import viteConfig from './vite.config.ts'

// Windows + Git Bash: `cd /d/...` gives the process a lower-case drive
// ("d:\..."), and Vitest then loads its own runtime twice (once per spelling
// of the path). Every file fails with "failed to find the current suite".
// Neither chdir nor an explicit root fixes it, because the CLI is already
// loaded by then, so stop early with a message that says what to do.
if (process.platform === 'win32' && /^[a-z]:/.test(process.cwd())) {
  throw new Error(
    `Vitest needs an upper-case drive letter on Windows (cwd is ${process.cwd()}). ` +
      'In Git Bash: cd /D/... instead of cd /d/...; PowerShell and cmd are fine.',
  )
}

// Unit and component tests (Vitest + React Testing Library, jsdom).
// Browser E2E tests live in e2e/ and run with Playwright (playwright.config.ts).
export default mergeConfig(
  viteConfig,
  defineConfig({
    test: {
      environment: 'jsdom',
      include: ['src/**/*.test.{ts,tsx}'],
      setupFiles: ['src/test/setup.ts'],
      restoreMocks: true,
    },
  }),
)

import {defineConfig, devices} from '@playwright/test';

import {BASE_URL, ENV_FACTOR} from './e2e/fixtures/auth';
import {STORAGE_STATE} from './e2e/paths';

// E2E browser tests for the Argo Workflows UI. These run against a live stack
// started with `make start` (see docs/running-locally.md), not as part of
// `yarn test` (Jest). Timeouts scale by E2E_ENV_FACTOR to absorb the resource
// contention the Go e2e suite also compensates for.
const isCI = !!process.env.CI;

export default defineConfig({
    testDir: './e2e/tests',
    globalSetup: require.resolve('./e2e/global-setup'),
    outputDir: './test-results',
    fullyParallel: true,
    forbidOnly: isCI,
    retries: isCI ? 2 : 0,
    workers: isCI ? 2 : undefined,
    timeout: 60_000 * ENV_FACTOR,
    expect: {timeout: 15_000 * ENV_FACTOR},
    reporter: isCI ? [['list'], ['html', {open: 'never'}], ['github']] : [['list'], ['html', {open: 'never'}]],
    use: {
        baseURL: BASE_URL,
        storageState: STORAGE_STATE,
        actionTimeout: 15_000 * ENV_FACTOR,
        navigationTimeout: 30_000 * ENV_FACTOR,
        trace: 'retain-on-failure',
        video: 'retain-on-failure',
        screenshot: 'only-on-failure'
    },
    projects: [{name: 'chromium', use: {...devices['Desktop Chrome']}}]
});

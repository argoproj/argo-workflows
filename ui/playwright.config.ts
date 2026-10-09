import {defineConfig, devices, ReporterDescription} from '@playwright/test';

import {BASE_URL, ENV_FACTOR} from './e2e/fixtures/auth';
import {STORAGE_STATE} from './e2e/paths';

// E2E browser tests for the Argo Workflows UI. These run against a live stack
// started with `make start` (see docs/running-locally.md), not as part of
// `yarn test` (Jest). Timeouts scale by E2E_ENV_FACTOR to absorb the resource
// contention the Go e2e suite also compensates for.
const isCI = !!process.env.CI;
// CI runs the suite several times per job (production bundle, dev server, dev
// server under a base href); E2E_RUN names the pass so each keeps its own
// results and report under the usual directories.
const run = process.env.E2E_RUN ? `/${process.env.E2E_RUN}` : '';
const html: ReporterDescription = ['html', {open: 'never', outputFolder: `playwright-report${run}`}];

export default defineConfig({
    testDir: './e2e/tests',
    globalSetup: require.resolve('./e2e/global-setup'),
    outputDir: `./test-results${run}`,
    fullyParallel: true,
    forbidOnly: isCI,
    retries: isCI ? 2 : 0,
    workers: isCI ? 2 : undefined,
    timeout: 60_000 * ENV_FACTOR,
    expect: {timeout: 15_000 * ENV_FACTOR},
    reporter: isCI ? [['list'], html, ['github']] : [['list'], html],
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

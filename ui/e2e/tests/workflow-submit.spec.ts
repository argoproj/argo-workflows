import {ENV_FACTOR} from '../fixtures/auth';
import {expect, test} from '../fixtures/test';

test('submits the example workflow from the UI and it runs to completion', async ({api, workflowListPage, workflowDetailsPage}) => {
    await workflowListPage.goto();
    await workflowListPage.submitExampleWorkflow();

    // The example manifest names itself, so the name is only knowable once the UI
    // has redirected to the new workflow's details page.
    const name = await workflowDetailsPage.currentWorkflowName();
    api.track(name);

    // Unlike the seeded tests, this one necessarily starts from a workflow that
    // has not run yet, so allow a full pull-and-run before it reports Succeeded.
    // Keep this comfortably under playwright.config.ts's per-test timeout (also
    // scaled by E2E_ENV_FACTOR), or the test dies before the assertion reports.
    await workflowDetailsPage.openTab('Summary');
    await expect(workflowDetailsPage.summaryAttribute('Status')).toContainText('Succeeded', {timeout: 45_000 * ENV_FACTOR});
});

import {ENV_FACTOR} from '../fixtures/auth';
import {expect, test} from '../fixtures/test';
import {failingWorkflow, sleepWorkflow} from '../fixtures/workflows';

test('terminates a running workflow from the details page', async ({api, confirmDialog, workflowDetailsPage}) => {
    const name = await api.submitWorkflow(sleepWorkflow(60));
    await api.waitForPhase(name, 'Running');

    await workflowDetailsPage.goto(name);
    // Terminate is only offered while the workflow is running, so finding the
    // button is itself an assertion that the page reflects the live phase.
    await workflowDetailsPage.operation('Terminate').click();
    await confirmDialog.ok.click();

    await workflowDetailsPage.openTab('Summary');
    // Terminating means killing a pod, so give it more than the default expect
    // timeout — but stay well inside the per-test timeout, which scales too.
    await expect(workflowDetailsPage.summaryAttribute('Status')).toContainText('Failed', {timeout: 30_000 * ENV_FACTOR});
});

test('retries a failed workflow from the details page', async ({api, workflowDetailsPage}) => {
    const name = await api.submitWorkflow(failingWorkflow());
    await api.waitForPhase(name, 'Failed');
    const failedAt = (await api.getWorkflow(name)).status.finishedAt;

    await workflowDetailsPage.goto(name);
    await workflowDetailsPage.operation('Retry').click();

    // Retry opens a panel of retry options rather than a confirm dialog.
    await expect(workflowDetailsPage.panel.getByRole('heading', {name: 'Retry Workflow'})).toBeVisible();
    await workflowDetailsPage.panel.getByRole('button', {name: 'Retry'}).click();
    await expect(workflowDetailsPage.panel).toBeHidden();

    // A retry reuses the workflow's name, so the observable effect is that the
    // previous run's finish time is replaced. Watching for a `Running` phase in
    // the UI instead would race this workflow's (immediate) second failure.
    await expect.poll(async () => (await api.getWorkflow(name)).status.finishedAt, {timeout: 20_000 * ENV_FACTOR}).not.toBe(failedAt);
});

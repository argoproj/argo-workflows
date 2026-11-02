import {expect, test} from '../fixtures/test';
import {echoWorkflow} from '../fixtures/workflows';

test('resubmits a workflow from the details page', async ({api, workflowDetailsPage}) => {
    const name = await api.submitWorkflow(echoWorkflow());
    await api.waitForPhase(name, 'Succeeded');

    await workflowDetailsPage.goto(name);
    await workflowDetailsPage.operation('Resubmit').click();

    // Resubmit opens a panel of resubmit options rather than a confirm dialog.
    await expect(workflowDetailsPage.panel.getByRole('heading', {name: 'Resubmit Workflow'})).toBeVisible();
    await workflowDetailsPage.panel.getByRole('button', {name: 'Resubmit'}).click();

    // The resubmit creates a separate workflow and redirects to it.
    const resubmitted = await workflowDetailsPage.waitForRedirectFrom(name);
    api.track(resubmitted);
    await expect(workflowDetailsPage.operation('Resubmit')).toBeVisible();
});

test('deletes a workflow from the list', async ({api, confirmDialog, workflowListPage}) => {
    const name = await api.submitWorkflow(echoWorkflow());
    await api.waitForPhase(name, 'Succeeded');

    await workflowListPage.goto();
    await workflowListPage.filterByName(name);
    await expect(workflowListPage.row(name)).toBeVisible();

    // Selecting a row reveals the bulk toolbar; its Delete acts on the selection.
    await workflowListPage.rowCheckbox(name).check();
    await workflowListPage.bulkAction('Delete').click();
    await confirmDialog.ok.click();

    await expect(workflowListPage.row(name)).toBeHidden();
});

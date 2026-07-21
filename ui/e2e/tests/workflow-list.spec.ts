import {expect, test} from '../fixtures/test';
import {echoWorkflow} from '../fixtures/workflows';

test('lists submitted workflows, filters by name, and opens details', async ({api, page, workflowListPage}) => {
    // Seed two workflows and wait for terminal state so the list is not racing the controller.
    const [wanted, other] = await Promise.all([api.submitWorkflow(echoWorkflow('hello e2e')), api.submitWorkflow(echoWorkflow('other e2e'))]);
    await Promise.all([api.waitForPhase(wanted, 'Succeeded'), api.waitForPhase(other, 'Succeeded')]);

    await workflowListPage.goto();

    // Both rows render, each with a phase icon.
    await expect(workflowListPage.row(wanted)).toBeVisible();
    await expect(workflowListPage.row(other)).toBeVisible();
    await expect(workflowListPage.rowStatusIcon(wanted)).toBeVisible();

    // Name filter narrows the list to the matching workflow.
    await workflowListPage.filterByName(wanted);
    await expect(workflowListPage.row(wanted)).toBeVisible();
    await expect(workflowListPage.row(other)).toBeHidden();

    // Clicking the row navigates to its details page.
    await workflowListPage.openWorkflow(wanted);
    await expect(page).toHaveURL(new RegExp(`/workflows/${api.namespace}/${wanted}`));
});

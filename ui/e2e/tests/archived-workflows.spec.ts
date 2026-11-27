import {expect, test} from '../fixtures/test';
import {echoWorkflow} from '../fixtures/workflows';

// With the workflow archive enabled, the workflow list merges live and archived
// workflows, so a completed workflow deleted from the cluster is still listed,
// served from the archive. Its details load from the archive too: getting a
// workflow the cluster no longer has falls back to it.
test('lists a workflow deleted from the cluster as archived and opens it from the archive', async ({api, page, workflowListPage, workflowDetailsPage}) => {
    test.skip(!(await api.archiveEnabled()), 'needs the workflow archive (the mysql or postgres profile)');

    const name = await api.submitWorkflow(echoWorkflow());
    // With the archive on, this also waits for the workflow to be archived.
    await api.waitForPhase(name, 'Succeeded');
    const {uid} = (await api.getWorkflow(name)).metadata;
    await api.deleteWorkflowFromCluster(name);

    // From here on the workflow exists only in the archive, so everything the
    // page shows about it was served from there.
    await workflowListPage.goto();
    await workflowListPage.filterByName(name);
    await expect(workflowListPage.rowArchived(name)).toHaveText('true');

    // The row links by uid, which is how the details page finds an archived workflow.
    await workflowListPage.openWorkflow(name);
    await expect(page).toHaveURL(new RegExp(`[?&]uid=${uid}`));
    await workflowDetailsPage.openTab('Summary');
    await expect(workflowDetailsPage.summaryAttribute('Status')).toContainText('Succeeded');
});

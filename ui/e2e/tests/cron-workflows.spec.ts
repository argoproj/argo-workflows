import {expect, test} from '../fixtures/test';
import {cronWorkflow} from '../fixtures/workflows';

test('creates a cron workflow from the UI and lists it with its schedule', async ({api, cronWorkflowPage}) => {
    await cronWorkflowPage.gotoList();
    await cronWorkflowPage.createExample();

    // The example manifest names itself, so the name is only knowable once the
    // UI has redirected to the new cron workflow's details page. Its schedule is
    // every minute, so it may spawn a workflow before teardown deletes it;
    // Kubernetes garbage-collects that along with it.
    const name = await cronWorkflowPage.currentName();
    api.track(name, 'cronWorkflow');

    await cronWorkflowPage.gotoList();
    const row = cronWorkflowPage.rowContainer(name);
    await expect(row).toContainText('* * * * *');
    // The cron expression's description, rendered by cronstrue.
    await expect(row).toContainText('Every minute');
});

test('suspends and resumes a cron workflow from the details page', async ({api, cronWorkflowPage}) => {
    const name = await api.createCronWorkflow(cronWorkflow());
    const suspended = async () => (await api.getCronWorkflow(name)).spec.suspend ?? false;

    await cronWorkflowPage.goto(name);

    // The toolbar offers exactly one of Suspend and Resume, whichever applies,
    // and flips only once the server has answered, so the stored state can be
    // checked straight after.
    await cronWorkflowPage.operation('Suspend').click();
    await expect(cronWorkflowPage.operation('Resume')).toBeVisible();
    await expect(cronWorkflowPage.operation('Suspend')).toBeHidden();
    expect(await suspended()).toBe(true);

    await cronWorkflowPage.operation('Resume').click();
    await expect(cronWorkflowPage.operation('Suspend')).toBeVisible();
    await expect(cronWorkflowPage.operation('Resume')).toBeHidden();
    expect(await suspended()).toBe(false);
});

import {ENV_FACTOR} from '../fixtures/auth';
import {expect, test} from '../fixtures/test';
import {echoWorkflow} from '../fixtures/workflows';

test('shows the container logs of a completed workflow', async ({api, workflowDetailsPage, workflowLogsPanel}) => {
    const message = 'hello from the logs test';
    const name = await api.submitWorkflow(echoWorkflow(message, 'e2e-logs-'));
    await api.waitForPhase(name, 'Succeeded');

    await workflowDetailsPage.goto(name);
    await workflowDetailsPage.operation('Logs').click();
    await expect(workflowLogsPanel.heading).toBeVisible();

    // The panel opens on "All" pods, so each line is prefixed with its pod name;
    // a single-step workflow's only pod is named after the workflow. The stream
    // is served over SSE from the (completed, not garbage-collected) pod.
    await expect.poll(() => workflowLogsPanel.text(), {timeout: 30_000 * ENV_FACTOR}).toContain(`${name}: ${message}`);
});

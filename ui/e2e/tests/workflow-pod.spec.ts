import {ENV_FACTOR} from '../fixtures/auth';
import {expect, test} from '../fixtures/test';
import {dagWorkflow} from '../fixtures/workflows';

test('shows the pod of a node from the node-info panel', async ({api, workflowDetailsPage, workflowPodPanel}) => {
    const name = await api.submitWorkflow(dagWorkflow('e2e-pod-'));
    await api.waitForPhase(name, 'Succeeded');

    await workflowDetailsPage.goto(name);
    await workflowDetailsPage.openNode('process-a');
    const podName = (await workflowDetailsPage.nodeAttribute('POD NAME').innerText()).trim();
    // The accessible name starts with the icon glyph, so exact matching fails, and a substring match would also hit "Pod Link".
    await workflowDetailsPage.nodeInfo.getByRole('button', {name: /POD$/}).click();

    // The pod of a completed node is not garbage-collected, so the watch's initial ADDED event carries it.
    await expect(workflowPodPanel.editor).toContainText(podName, {timeout: 30_000 * ENV_FACTOR});
});

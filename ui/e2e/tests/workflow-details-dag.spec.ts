import {expect, test} from '../fixtures/test';
import {DAG_TASKS, dagWorkflow} from '../fixtures/workflows';

test('renders every DAG node and opens the node-info panel', async ({api, workflowDetailsPage}) => {
    const name = await api.submitWorkflow(dagWorkflow());
    await api.waitForPhase(name, 'Succeeded');

    await workflowDetailsPage.goto(name);

    for (const task of DAG_TASKS) {
        await expect(workflowDetailsPage.node(task)).toBeVisible();
    }

    // Clicking a node opens the side panel on its summary. A DAG task's node name
    // is `<workflow>.<task>`, which is what distinguishes it from its siblings.
    await workflowDetailsPage.openNode('process-a');
    await expect(workflowDetailsPage.nodeAttribute('NAME')).toContainText(`${name}.process-a`);
    await expect(workflowDetailsPage.nodeAttribute('PHASE')).toContainText('Succeeded');
});

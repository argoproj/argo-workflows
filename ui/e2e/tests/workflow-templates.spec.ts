import {ENV_FACTOR} from '../fixtures/auth';
import {expect, test} from '../fixtures/test';
import {echoWorkflowTemplate, TEMPLATE_MESSAGE} from '../fixtures/workflows';

// Workflow templates and cluster workflow templates share their editor and
// submit panel, so the same journeys run against both.
const KINDS = [
    {kind: 'workflow template', clusterScoped: false, label: 'workflows.argoproj.io/workflow-template'},
    {kind: 'cluster workflow template', clusterScoped: true, label: 'workflows.argoproj.io/cluster-workflow-template'}
];

for (const {kind, clusterScoped, label} of KINDS) {
    test.describe(kind, () => {
        test.use({clusterScopedTemplates: clusterScoped});

        test(`lists a ${kind} and opens its details`, async ({api, page, workflowTemplatePage}) => {
            const name = await api.createWorkflowTemplate(echoWorkflowTemplate(clusterScoped), clusterScoped);

            await workflowTemplatePage.gotoList();
            await workflowTemplatePage.openTemplate(name);

            await expect(page.locator('.top-bar__breadcrumbs-last-item')).toHaveText(name);
            // The Spec tab renders from the fetched template, not the URL, so its
            // parameter proves the details loaded.
            await workflowTemplatePage.openTab('Spec');
            await expect(workflowTemplatePage.specParameter('message')).toHaveText(TEMPLATE_MESSAGE);
        });

        test(`submits a workflow from a ${kind} and it runs to completion`, async ({api, workflowTemplatePage, workflowDetailsPage}) => {
            const template = await api.createWorkflowTemplate(echoWorkflowTemplate(clusterScoped), clusterScoped);

            await workflowTemplatePage.goto(template);
            await workflowTemplatePage.submit();

            // The submit panel redirects to the new workflow, whose name the server generated.
            const name = await workflowDetailsPage.currentWorkflowName();
            api.track(name);
            expect((await api.getWorkflow(name)).metadata.labels[label]).toBe(template);

            // As in workflow-submit.spec.ts, this run starts from scratch, so allow a
            // full pull-and-run, inside the (also scaled) per-test timeout.
            await workflowDetailsPage.openTab('Summary');
            await expect(workflowDetailsPage.summaryAttribute('Status')).toContainText('Succeeded', {timeout: 45_000 * ENV_FACTOR});
        });
    });
}

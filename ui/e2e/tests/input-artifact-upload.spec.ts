import {bearer, ENV_FACTOR, NAMESPACE} from '../fixtures/auth';
import {expect, test} from '../fixtures/test';
import {artifactWorkflowTemplate, UPLOAD_CONTENT} from '../fixtures/workflows';

// The two kinds share the submit panel, so the same journey runs against both.
const KINDS = [
    {kind: 'workflow template', clusterScoped: false},
    {kind: 'cluster workflow template', clusterScoped: true}
];

const UPLOADED_KEY = new RegExp(`uploads/${NAMESPACE}/[0-9a-f-]{36}/input\\.txt`);

for (const {kind, clusterScoped} of KINDS) {
    test.describe(kind, () => {
        test.use({clusterScopedTemplates: clusterScoped});

        test(`uploads an input artifact from the submit panel and the workflow reads it`, async ({api, workflowTemplatePage, workflowDetailsPage}) => {
            const template = await api.createWorkflowTemplate(artifactWorkflowTemplate(clusterScoped), clusterScoped);

            await workflowTemplatePage.goto(template);
            await workflowTemplatePage.openSubmitPanel();
            await workflowTemplatePage.chooseArtifactFile('input.txt', UPLOAD_CONTENT);
            await expect(workflowTemplatePage.panel.getByText(new RegExp(`Uploaded: ${UPLOADED_KEY.source}`))).toBeVisible();
            await workflowTemplatePage.panel.getByRole('button', {name: 'Submit'}).click();

            const name = await workflowDetailsPage.currentWorkflowName();
            api.track(name);

            const workflow = await api.getWorkflow(name);
            const artifact = workflow.spec.arguments.artifacts.find((a: {name: string}) => a.name === 'input-artifact');
            expect(artifact.s3.key).toMatch(new RegExp(`^${UPLOADED_KEY.source}$`));

            // Success proves the file reached MinIO under that key; the run starts from scratch, so allow a full pull-and-run.
            await api.waitForPhase(name, 'Succeeded', 60_000);
        });
    });
}

test.describe('cluster workflow template namespace', () => {
    test.use({clusterScopedTemplates: true});

    test('discards the uploaded file when the namespace changes and submits no artifact override', async ({api, page, workflowTemplatePage}) => {
        const template = await api.createWorkflowTemplate(artifactWorkflowTemplate(true), true);

        await workflowTemplatePage.goto(template);
        await workflowTemplatePage.openSubmitPanel();
        await workflowTemplatePage.chooseArtifactFile('input.txt', UPLOAD_CONTENT);
        const panel = workflowTemplatePage.panel;
        await expect(panel.getByText(new RegExp(`Uploaded: ${UPLOADED_KEY.source}`))).toBeVisible();
        await expect(panel.getByText('Namespace changed. Upload the input artifacts again.')).toHaveCount(0);

        await workflowTemplatePage.namespaceInput.fill('other');
        await expect(panel.getByText(/Uploaded: uploads\//)).toHaveCount(0);
        await expect(panel.getByText(/uploaded successfully/i)).toHaveCount(0);
        await expect(panel.getByText('Namespace changed. Upload the input artifacts again.')).toBeVisible();

        // Stub the submit call: the point is what the UI sends, and `other` is not a namespace to create workflows in.
        let submitted: {namespace: string; resourceKind: string; submitOptions: {artifacts?: string[]}} | undefined;
        await page.route(/\/api\/v1\/workflows\/[^/]+\/submit$/, async route => {
            submitted = route.request().postDataJSON();
            await route.fulfill({status: 400, contentType: 'application/json', body: JSON.stringify({code: 3, message: 'stubbed by the test'})});
        });
        await panel.getByRole('button', {name: 'Submit'}).click();
        await expect.poll(() => submitted, {timeout: 15_000 * ENV_FACTOR}).toBeDefined();
        expect(submitted!.namespace).toBe('other');
        expect(submitted!.resourceKind).toBe('ClusterWorkflowTemplate');
        expect(submitted!.submitOptions.artifacts).toBeUndefined();
    });

    test('is not enough on its own: the server rejects an upload key from another namespace at submit', async ({api, request}) => {
        const template = await api.createWorkflowTemplate(artifactWorkflowTemplate(true), true);

        const res = await request.post(`api/v1/workflows/${NAMESPACE}/submit`, {
            headers: {Authorization: bearer()},
            data: {
                namespace: NAMESPACE,
                resourceKind: 'ClusterWorkflowTemplate',
                resourceName: template,
                submitOptions: {artifacts: ['input-artifact=uploads/other/00000000-0000-4000-8000-000000000000/input.txt']}
            }
        });

        expect(res.status()).toBe(400);
        expect((await res.json()).message).toContain(`must start with "uploads/${NAMESPACE}/"`);
    });

    test('does not offer the upload before the namespace is known', async ({api, page, workflowTemplatePage}) => {
        const template = await api.createWorkflowTemplate(artifactWorkflowTemplate(true), true);
        // Hold the server info back, as a slow workflow list would, so the details page does not know its namespace yet.
        await page.route('**/api/v1/info', async route => {
            await new Promise(resolve => setTimeout(resolve, 5_000));
            await route.continue();
        });

        await page.goto(`cluster-workflow-templates/${template}?sidePanel=true`);
        await expect(workflowTemplatePage.panel.getByRole('heading', {name: 'Submit Workflow'})).toBeVisible();
        await expect(workflowTemplatePage.namespaceInput).toHaveValue('');
        await expect(workflowTemplatePage.panel.locator('input[type="file"]')).toHaveCount(0);

        await expect(workflowTemplatePage.namespaceInput).not.toHaveValue('', {timeout: 15_000 * ENV_FACTOR});
        await expect(workflowTemplatePage.panel.locator('input[type="file"]')).toHaveCount(1);
    });
});

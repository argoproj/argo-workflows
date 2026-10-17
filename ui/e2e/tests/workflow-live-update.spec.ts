import {ENV_FACTOR} from '../fixtures/auth';
import {expect, test} from '../fixtures/test';
import {sleepWorkflow} from '../fixtures/workflows';

// The only test that watches a phase transition in the browser: it covers the
// server-sent-events watch (api/v1/workflow-events) that keeps the details page
// current, which nothing else exercises — the other tests seed workflows in a
// terminal phase first.
test('details page follows a running workflow to completion without reloading', async ({api, page, workflowDetailsPage}) => {
    // Long enough that the page reliably catches Running before the pod exits;
    // short enough that Succeeded arrives well inside the per-test timeout.
    const name = await api.submitWorkflow(sleepWorkflow(20, 'e2e-live-'));

    await workflowDetailsPage.goto(name);
    await workflowDetailsPage.openTab('Summary');
    const status = workflowDetailsPage.summaryAttribute('Status');
    await expect(status).toContainText('Running', {timeout: 30_000 * ENV_FACTOR});

    // Mark this document so a reload — which would produce a fresh window
    // without the mark — can be told apart from a live update.
    await page.evaluate(() => {
        (window as any).__e2eSameDocument = true;
    });

    await expect(status).toContainText('Succeeded', {timeout: 45_000 * ENV_FACTOR});
    expect(await page.evaluate(() => (window as any).__e2eSameDocument)).toBe(true);
});

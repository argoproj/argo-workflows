import {NAMESPACE} from '../fixtures/auth';
import {expect, test} from '../fixtures/test';
import {echoWorkflow, withLabels} from '../fixtures/workflows';

// The reports page charts the workflows matching a namespace and label. It reads
// the ordinary workflow list, so it works with or without the archive (with it,
// completed workflows come from the archive rather than the cluster).
test('reports on the workflows carrying a label', async ({api, page}) => {
    // A label only this test's workflows carry, so the report covers exactly them.
    const label = `e2e-report=${Math.random().toString(36).slice(2, 10)}`;
    const [key, value] = label.split('=');
    const names = await Promise.all([api.submitWorkflow(withLabels(echoWorkflow(), {[key]: value})), api.submitWorkflow(withLabels(echoWorkflow(), {[key]: value}))]);
    await Promise.all(names.map(name => api.waitForPhase(name, 'Succeeded')));

    await page.goto(`reports/${NAMESPACE}?labels=${encodeURIComponent(label)}`);

    // Each chart (duration, then resources) draws to a canvas, so the bars are
    // not in the DOM; the footnote counting the charted workflows is.
    await expect(page.getByText(`${names.length} records.`)).toBeVisible();
    await expect(page.locator('.white-box canvas')).toHaveCount(2);
});

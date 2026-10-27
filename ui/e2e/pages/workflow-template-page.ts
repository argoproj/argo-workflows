import {expect, Locator, Page} from '@playwright/test';

import {NAMESPACE} from '../fixtures/auth';

// Page object for the list and details pages of workflow templates
// (ui/src/workflow-templates) or, when `clusterScoped`, of cluster workflow
// templates (ui/src/cluster-workflow-templates). Both kinds share the editor
// and the submit panel, and each page is small, so one object covers all four.
export class WorkflowTemplatePage {
    readonly toolbar: Locator;
    readonly panel: Locator;
    /** The list page's path, which every details path extends with the template's name. */
    private readonly listPath: string;

    constructor(
        private readonly page: Page,
        private readonly clusterScoped = false
    ) {
        // The action menu row under the breadcrumbs; see WorkflowDetailsPage.
        this.toolbar = page.locator('.top-bar.row');
        this.panel = page.locator('.sliding-panel--opened .sliding-panel__body');
        this.listPath = clusterScoped ? 'cluster-workflow-templates' : `workflow-templates/${NAMESPACE}`;
    }

    async gotoList(): Promise<void> {
        await this.page.goto(this.listPath);
    }

    async goto(name: string): Promise<void> {
        await this.page.goto(`${this.listPath}/${name}`);
    }

    /** The row's name link, matched by its details-page href. */
    row(name: string): Locator {
        return this.page.locator(`a[href$="/${this.listPath}/${name}"]`);
    }

    async openTemplate(name: string): Promise<void> {
        await this.row(name).click();
        await expect(this.page).toHaveURL(new RegExp(`/${this.listPath}/${name}`));
    }

    /** A toolbar action on the details page (Submit, Update, Delete, ...). */
    operation(title: string): Locator {
        return this.toolbar.getByRole('button', {name: title});
    }

    async openTab(title: 'Manifest' | 'Spec' | 'MetaData' | 'Workflow MetaData' | 'Graph'): Promise<void> {
        await this.page.locator('.tabs__nav').getByText(title, {exact: true}).click();
    }

    /**
     * The value shown for a workflow parameter in the Spec tab. KeyValueEditor
     * renders each pair as a row of name and value columns, followed by an
     * empty row of inputs for adding another.
     */
    specParameter(name: string): Locator {
        return this.page
            .locator('.white-box__details-row')
            .filter({has: this.page.locator('.columns.small-4', {hasText: new RegExp(`^${name}$`)})})
            .locator('.columns.small-6');
    }

    /**
     * Submits a workflow from the template into the suite's namespace, taking
     * the submit panel's other defaults.
     */
    async submit(): Promise<void> {
        await this.operation('Submit').click();
        await expect(this.panel.getByRole('heading', {name: 'Submit Workflow'})).toBeVisible();
        if (this.clusterScoped) {
            // A cluster template has no namespace of its own, so the panel asks
            // for one. It is pre-filled once the details page has fetched the
            // server info (falling back to `default`); wait for that before
            // replacing it, or the late default would overwrite what we type.
            const namespace = this.panel.locator('xpath=.//label[normalize-space()="Namespace"]/following-sibling::input');
            await expect(namespace).not.toHaveValue('');
            await namespace.fill(NAMESPACE);
            await expect(namespace).toHaveValue(NAMESPACE);
        }
        await this.panel.getByRole('button', {name: 'Submit'}).click();
    }
}

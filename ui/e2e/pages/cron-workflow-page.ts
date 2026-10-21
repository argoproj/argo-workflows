import {expect, Locator, Page} from '@playwright/test';

import {NAMESPACE} from '../fixtures/auth';

// Page object for the cron workflow list and details pages
// (ui/src/cron-workflows). Each is small, so one object covers both.
export class CronWorkflowPage {
    readonly toolbar: Locator;
    readonly panel: Locator;

    constructor(private readonly page: Page) {
        // The action menu row under the breadcrumbs; see WorkflowDetailsPage.
        this.toolbar = page.locator('.top-bar.row');
        this.panel = page.locator('.sliding-panel--opened .sliding-panel__body');
    }

    async gotoList(): Promise<void> {
        await this.page.goto(`cron-workflows/${NAMESPACE}`);
    }

    async goto(name: string): Promise<void> {
        await this.page.goto(`cron-workflows/${NAMESPACE}/${name}`);
    }

    /** The row's name link, matched by its details-page href. */
    row(name: string): Locator {
        return this.page.locator(`a[href$="/cron-workflows/${NAMESPACE}/${name}"]`);
    }

    /** The whole list row, whose cells (schedule, its description, ...) carry no names of their own. */
    rowContainer(name: string): Locator {
        return this.page.locator('.cron-workflows-list__row-container').filter({has: this.row(name)});
    }

    /** A toolbar action (Create New Cron Workflow on the list; Submit, Suspend, Resume, ... on details). */
    operation(title: string): Locator {
        return this.toolbar.getByRole('button', {name: title});
    }

    /**
     * Creates a cron workflow from the list page's create panel, taking the
     * pre-filled example manifest (ui/src/shared/examples.ts) as offered: the
     * editor is Monaco, so tests must not type into it.
     */
    async createExample(): Promise<void> {
        await this.operation('Create New Cron Workflow').click();
        await this.panel.getByRole('button', {name: 'Create'}).click();
    }

    /** The name of the cron workflow whose details are shown; the example's is random. */
    async currentName(): Promise<string> {
        const pattern = new RegExp(`/cron-workflows/${NAMESPACE}/([^/?#]+)`);
        await expect(this.page).toHaveURL(pattern);
        const match = pattern.exec(this.page.url());
        if (!match) {
            throw new Error(`not on a cron workflow details page: ${this.page.url()}`);
        }
        return match[1];
    }
}

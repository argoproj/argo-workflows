import {expect, Locator, Page} from '@playwright/test';

import {NAMESPACE} from '../fixtures/auth';

// Page object for the workflows list (ui/src/workflows/components/workflows-list).
export class WorkflowListPage {
    readonly nameFilter: Locator;

    constructor(private readonly page: Page) {
        // The name filter is an unlabelled InputFilter; workflow-filters.tsx tags
        // its container with data-testid="workflow-name-filter".
        this.nameFilter = page.getByTestId('workflow-name-filter').locator('input');
    }

    async goto(): Promise<void> {
        await this.page.goto('workflows');
    }

    /** The row's clickable name link, matched by its stable details-page href. */
    row(name: string): Locator {
        return this.page.locator(`a[href*="/workflows/${NAMESPACE}/${name}"]`);
    }

    /** The phase icon within a given row's status cell. */
    rowStatusIcon(name: string): Locator {
        // Scope to the row container holding this workflow's name link, then its
        // status cell (a stable data-testid), rather than walking the row markup.
        return this.page
            .locator('.workflows-list__row-container')
            .filter({has: this.row(name)})
            .getByTestId('workflow-status')
            .locator('i');
    }

    async filterByName(name: string): Promise<void> {
        // InputFilter only commits the filter (a list refetch) on Enter, and its
        // Enter handler reads React state rather than the input's live value. Wait
        // until the controlled input reflects `name` — i.e. React has flushed the
        // fill() update into state — before pressing Enter, else a stale/empty
        // value can be committed and the list won't filter.
        await this.nameFilter.fill(name);
        await expect(this.nameFilter).toHaveValue(name);
        await this.nameFilter.press('Enter');
    }

    async openWorkflow(name: string): Promise<void> {
        await this.row(name).click();
        await expect(this.page).toHaveURL(new RegExp(`/workflows/${NAMESPACE}/${name}`));
    }
}

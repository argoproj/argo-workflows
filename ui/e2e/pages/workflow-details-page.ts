import {expect, Locator, Page} from '@playwright/test';

import {NAMESPACE} from '../fixtures/auth';

/** Escapes `value` for embedding in a RegExp; workflow names contain `.` and `-`. */
function quote(value: string): string {
    return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

// Page object for the workflow details page (ui/src/workflows/components/workflow-details):
// its operation toolbar, the DAG graph and the node-info side panel.
export class WorkflowDetailsPage {
    readonly toolbar: Locator;
    readonly graph: Locator;
    readonly nodeInfo: Locator;
    readonly panel: Locator;

    constructor(private readonly page: Page) {
        // argo-ui's TopBar renders the breadcrumb bar as `top-bar` and the action
        // menu below it as `top-bar row`, so both classes are needed to pick out
        // the operation buttons.
        this.toolbar = page.locator('.top-bar.row');
        this.graph = page.locator('.graph');
        this.nodeInfo = page.locator('.workflow-node-info');
        this.panel = page.locator('.sliding-panel--opened .sliding-panel__body');
    }

    async goto(name: string): Promise<void> {
        await this.page.goto(`workflows/${NAMESPACE}/${name}`);
    }

    /** The name of the workflow currently shown — the only place a UI-created name surfaces. */
    async currentWorkflowName(): Promise<string> {
        const pattern = new RegExp(`/workflows/${NAMESPACE}/([^/?#]+)`);
        await expect(this.page).toHaveURL(pattern);
        const match = pattern.exec(this.page.url());
        if (!match) {
            throw new Error(`not on a workflow details page: ${this.page.url()}`);
        }
        return match[1];
    }

    /**
     * Waits for an operation that creates a *new* workflow (resubmit, submit) to
     * redirect away from `name`, then returns the new workflow's name.
     */
    async waitForRedirectFrom(name: string): Promise<string> {
        await expect(this.page).not.toHaveURL(new RegExp(`/workflows/${NAMESPACE}/${quote(name)}(?:[?#]|$)`));
        return this.currentWorkflowName();
    }

    /**
     * A toolbar operation button (Retry, Resubmit, Terminate, Delete, ...).
     * WorkflowOperationsMap only offers the operations that are legal for the
     * current phase, so these appear and disappear as a workflow progresses.
     */
    operation(title: string): Locator {
        return this.toolbar.getByRole('button', {name: title});
    }

    async openTab(title: 'Summary' | 'Events' | 'Timeline' | 'Workflow'): Promise<void> {
        await this.page.locator(`.workflow-details__topbar-buttons a[title="${title}"]`).click();
    }

    /** The value cell of a row in the Summary tab's attribute table, e.g. `Status`. */
    summaryAttribute(title: string): Locator {
        return this.page
            .locator('.white-box__details-row')
            .filter({has: this.page.getByText(title, {exact: true})})
            .locator('.columns.small-9');
    }

    /**
     * A node in the DAG. Each node is an SVG group holding a `<title>` of
     * `"<nodeId> (<label>)"` alongside the clickable inner `g.node`. Anchoring on
     * the bracketed label keeps this independent of the generated node id and of
     * the line-wrapping graph-panel applies to the visible label.
     */
    node(label: string): Locator {
        return this.graph
            .locator('svg > g > g')
            .filter({has: this.page.locator('title', {hasText: new RegExp(`\\(${quote(label)}\\)$`)})})
            .locator('g.node');
    }

    async openNode(label: string): Promise<void> {
        await this.node(label).click();
        await expect(this.nodeInfo).toBeVisible();
    }

    /**
     * The value cell for `title` in the node-info summary grid. AttributeRows
     * renders each pair as two sibling `<div>`s with nothing to tell them apart,
     * so the value is addressed as the label's next sibling. Only the selected
     * tab's content is mounted, so titles shared with other tabs stay unambiguous.
     */
    nodeAttribute(title: string): Locator {
        return this.nodeInfo.locator(`xpath=.//div[normalize-space()="${title}"]/following-sibling::div[1]`);
    }
}

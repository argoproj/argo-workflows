import {Locator, Page} from '@playwright/test';

export class WorkflowPodPanel {
    readonly editor: Locator;

    constructor(page: Page) {
        // Monaco does not expose rendered text as a control value, so query its `.view-lines` layer.
        this.editor = page.locator('.sliding-panel--opened .monaco-editor .view-lines');
    }
}

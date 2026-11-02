import {Locator, Page} from '@playwright/test';

// The app-wide confirmation popup (argo-ui's PopupManager.confirm), which gates
// both the workflow list's bulk actions and the details page's toolbar
// operations. The buttons carry no accessible id beyond their text.
export class ConfirmDialog {
    readonly container: Locator;
    readonly ok: Locator;
    readonly cancel: Locator;

    constructor(page: Page) {
        this.container = page.locator('.popup-container');
        this.ok = this.container.getByRole('button', {name: 'OK'});
        this.cancel = this.container.getByRole('button', {name: 'Cancel'});
    }
}

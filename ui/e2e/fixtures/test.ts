import {test as base} from '@playwright/test';

import {ConfirmDialog} from '../pages/confirm-dialog';
import {LoginPage} from '../pages/login-page';
import {WorkflowDetailsPage} from '../pages/workflow-details-page';
import {WorkflowListPage} from '../pages/workflow-list-page';
import {ApiClient} from './api';

interface Fixtures {
    api: ApiClient;
    confirmDialog: ConfirmDialog;
    loginPage: LoginPage;
    workflowDetailsPage: WorkflowDetailsPage;
    workflowListPage: WorkflowListPage;
}

export const test = base.extend<Fixtures>({
    // Playwright ignores uncaught exceptions in the page unless they happen to
    // interrupt an action, so bugs like argoproj/argo-workflows#16491 (userinfo
    // 401 throwing on the login page) pass silently. Collect them and fail the
    // test at teardown instead.
    page: async ({page}, use) => {
        const errors: Error[] = [];
        page.on('pageerror', err => errors.push(err));
        await use(page);
        if (errors.length > 0) {
            throw new Error(`uncaught exception(s) in page:\n\n${errors.map(err => err.stack ?? String(err)).join('\n\n')}`);
        }
    },
    api: async ({request}, use) => {
        const api = new ApiClient(request);
        await use(api);
        await api.cleanup();
    },
    confirmDialog: async ({page}, use) => {
        await use(new ConfirmDialog(page));
    },
    loginPage: async ({page}, use) => {
        await use(new LoginPage(page));
    },
    workflowDetailsPage: async ({page}, use) => {
        await use(new WorkflowDetailsPage(page));
    },
    workflowListPage: async ({page}, use) => {
        await use(new WorkflowListPage(page));
    }
});

export {expect} from '@playwright/test';

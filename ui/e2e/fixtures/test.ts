import {test as base} from '@playwright/test';

import {LoginPage} from '../pages/login-page';
import {WorkflowListPage} from '../pages/workflow-list-page';
import {ApiClient} from './api';

interface Fixtures {
    api: ApiClient;
    loginPage: LoginPage;
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
    loginPage: async ({page}, use) => {
        await use(new LoginPage(page));
    },
    workflowListPage: async ({page}, use) => {
        await use(new WorkflowListPage(page));
    }
});

export {expect} from '@playwright/test';

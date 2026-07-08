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

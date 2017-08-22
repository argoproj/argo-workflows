export const SEARCH_INDEX_TYPE = {
    SERVICES: 'services',
    TEMPLATES: 'templates',
    POLICIES: 'policies',
    PROJECTS: 'projects',
    APPLICATIONS: 'applications',
    DEPLOYMENT: 'deployments',
};

export class SearchIndex {
    key: string = '';
    type: string = '';
    value: string = '';

    constructor(data?) {
        if (data && typeof data === 'object') {
            Object.assign(this, data);
        }
    }
}

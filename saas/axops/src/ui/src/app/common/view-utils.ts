import { ViewPreferences } from '../model';
import { BranchesFiltersComponent } from './branches-filters/branches-filters.component';

export const ViewUtils = {
    scrollParent(el) {
        let regex = /(auto|scroll)/;

        let ps = $(el).parents();

        for (let i = 0; i < ps.length; i += 1) {
            let node = ps[i];
            let overflow = getComputedStyle(node, null).getPropertyValue('overflow') +
                getComputedStyle(node, null).getPropertyValue('overflow-y') +
                getComputedStyle(node, null).getPropertyValue('overflow-x');
            if (regex.test(overflow)) {
                return node;
            }
        }

        return document.body;
    },

    sanitizeRouteParams(params, ...updatedParams: any[]) {
        params = Object.assign(params, ...updatedParams);
        for (let key of Object.keys(params)) {
            if (params[key] === null || params[key] === undefined) {
                delete params[key];
            }
        }
        return params;
    },

    mapLabelsToList(labelsObject): string[] {
        let labels: string[] = [];
        for (let property in labelsObject) {
            if (labelsObject.hasOwnProperty(property)) {
                labels.push(`${property}: ${labelsObject[property]}`);
            }
        }
        return labels;
    },

    mapToKeyValue(object: {[key: string]: string}): {key: string, value: string}[] {
        let keyValueList = [];
        for (let property in object) {
            if (object.hasOwnProperty(property)) {
                keyValueList.push({key: property, value: object[property]});
            }
        }
        return keyValueList;
    },

    capitalizeFirstLetter(word: string) {
        return word.charAt(0).toUpperCase() + word.slice(1);
    },

    getBranchBreadcrumb(repo: string, branch: string, rootPage: string | { url: string, params: {} }, viewPreferences: ViewPreferences, leafPageTitle?: string, rootLink = false) {
        let rootUrl: string;
        let rootUrlParams = {};
        if (typeof rootPage === 'string') {
            rootUrl = rootPage as string;
        } else {
            rootUrl = rootPage.url;
            rootUrlParams = rootPage.params;
        }
        let breadcrumb: any[] = [{
            title: viewPreferences && viewPreferences.filterState.branches === 'my' ? 'My Favorite Branches' : 'All Branches',
            routerLink: ( repo || rootLink ) ? [rootUrl, Object.assign({}, rootUrlParams, { repo: '', branch: '' })] : null
        }];

        if (repo) {
            breadcrumb.push({
                title: BranchesFiltersComponent.parseRepoUrl(repo).name,
                routerLink: ( branch || rootLink ) ? [rootUrl, Object.assign({}, rootUrlParams, { repo, branch: '' })] : null
            });
        }

        if (branch) {
            breadcrumb.push({
                title: branch,
                routerLink: leafPageTitle ? [rootUrl, Object.assign({}, rootUrlParams, { repo, branch })] : null
            });
        }

        if (leafPageTitle) {
            breadcrumb.push({
                title: leafPageTitle
            });
        }

        return breadcrumb;
    },

    getSelectedRepoBranch(params, viewPreferences: ViewPreferences): [string, string] {
        let selectedRepo = params.hasOwnProperty('repo') ? decodeURIComponent(params['repo']) : viewPreferences.filterState.selectedRepo;
        let selectedBranch = params.hasOwnProperty('branch') ? decodeURIComponent(params['branch']) : viewPreferences.filterState.selectedBranch;
        return [selectedRepo, selectedBranch];
    }
};

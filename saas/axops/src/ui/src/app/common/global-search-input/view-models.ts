import { Pagination } from '../pagination/pagination';
import { TaskFieldNames, ApplicationFieldNames, TemplateFieldNames, DeploymentFieldNames, PolicyFieldNames } from '../../model';

export class InitFilters {
    repo?: string[] = [];
    branch?: string[] = [];
    pagination?: Pagination;
}

export class JobsFilters extends InitFilters {
    statuses?: string[] = [];
    authors?: string[] = [];
    artifact_tags?: string[] = [];
    templates?: string[] = [];
}

export class CommitsFilters extends InitFilters {
    authors?: string[] = [];
    committers?: string[] = [];
}

export class ArtifactTagsFilters extends InitFilters {
    authors?: string[] = [];
}

export class ApplicationsFilters {
    application_statuses?: string[] = [];
}

export class DeploymentsFilters {
    application_statuses?: string[] = [];
    app_name?: string[] = [];
}

export class GlobalSearchFilters {
    jobs: JobsFilters = new JobsFilters();
    commits: CommitsFilters = new CommitsFilters();
    artifact_tags: ArtifactTagsFilters = new ArtifactTagsFilters();
    applications: ApplicationsFilters = new ApplicationsFilters();
    deployments: DeploymentsFilters = new DeploymentsFilters();
    templates: InitFilters = new InitFilters();
}

export class SearchHistoryItem {
    key: string = '';
    addDate: number = 0;

    constructor(key: string, addDate: number) {
        this.key = key;
        this.addDate = addDate;
    }
}

export class SearchHistory {
    jobs: SearchHistoryItem[] = [];
    commits:  SearchHistoryItem[] = [];
    artifact_tags: SearchHistoryItem[] = [];
    applications: SearchHistoryItem[] = [];
    deployments: SearchHistoryItem[] = [];
    templates: SearchHistoryItem[] = [];
    policies: SearchHistoryItem[] = [];
    catalog: SearchHistoryItem[] = [];
    users: SearchHistoryItem[] = [];
}

export class GlobalSearchSetting {
    keepOpen: boolean;
    suppressBackRoute: boolean;
    searchString?: string;
    searchCategory?: string;
    filters?: GlobalSearchFilters;
    backRoute?: string;
    applyLocalSearchQuery?: (searchString: string, searchCategory: string) => void;
    hideSearchHistoryAndSuggestions?: boolean;
}

export const GLOBAL_SEARCH_TABS = {
    JOBS: {
        name: 'jobs',
        description: 'Jobs',
    },
    COMMITS: {
        name: 'commits',
        description: 'Commits',
        featureSets: ['full', 'limited_aws', 'limited'],
    },
    APPLICATIONS: {
        name: 'applications',
        description: 'Applications',
        featureSets: ['full', 'limited_aws'],
    },
    DEPLOYMENTS: {
        name: 'deployments',
        description: 'Deployments',
        featureSets: ['full', 'limited_aws'],
    },
    ARTIFACT_TAGS: {
        name: 'artifact_tags',
        description: 'Artifact Tags',
        featureSets: ['full', 'limited_aws', 'limited'],
    },
    TEMPLATES: {
        name: 'templates',
        description: 'Templates',
        featureSets: ['full', 'limited_aws', 'limited'],
    }
};

export const LOCAL_SEARCH_CATEGORIES = {
    POLICIES: {
        name: 'policies',
        description: 'Policies',
    },
    CATALOG: {
        name: 'catalog',
        description: 'Catalog',
    },
    USERS: {
        name: 'users',
        description: 'Users',
    },
};

export const GLOBAL_SEARCH_SUGGESTION_FIELDS_CONFIG = {
    services: [
        TaskFieldNames.name,
        TaskFieldNames.username,
        TaskFieldNames.status_string,
        TaskFieldNames.branch,
        TaskFieldNames.repo,
    ],
    commits: [

    ],
    applications: [
        ApplicationFieldNames.name,
    ],
    deployments: [
        DeploymentFieldNames.name,
    ],
    templates: [
        TemplateFieldNames.name,
        TemplateFieldNames.description,
        TemplateFieldNames.type,
        TemplateFieldNames.branch,
        TemplateFieldNames.repo,
    ],
    policies: [
        PolicyFieldNames.name,
        PolicyFieldNames.description,
        PolicyFieldNames.repo,
        PolicyFieldNames.branch,
        PolicyFieldNames.subtype,
        PolicyFieldNames.cost,
    ]
};

import { Component, Input, Output, EventEmitter, OnChanges, OnDestroy } from '@angular/core';
import { Subscription } from 'rxjs';

import { TASK_STATUSES } from '../../../pipes/statusToNumber.pipe';

import { FilterMultiSelect } from 'argo-ui-lib/src/components';
import { APPLICATION_STATUSES } from '../../../model';
import { GlobalSearchFilters } from '../../../common';
import { UsersService, ArtifactsService, BranchService, RepoService, ApplicationsService } from '../../../services';

const GLOBAL_SEARCH_FILTERS = {
    REPOS: 'repo',
    BRANCHES: 'branch',
    STATUSES: 'statuses',
    AUTHORS: 'authors',
    COMMITTERS: 'committers',
    ARTIFACT_TAGS: 'artifact_tags',
    APPLICATION_STATUSES: 'application_statuses',
    APP_NAME: 'app_name',
};

const GLOBAL_SEARCH_FILTER_CONFIG = {
    JOBS: [
        GLOBAL_SEARCH_FILTERS.REPOS,
        GLOBAL_SEARCH_FILTERS.BRANCHES,
        GLOBAL_SEARCH_FILTERS.STATUSES,
        GLOBAL_SEARCH_FILTERS.AUTHORS,
        GLOBAL_SEARCH_FILTERS.ARTIFACT_TAGS,
    ],
    COMMITS: [
        GLOBAL_SEARCH_FILTERS.REPOS,
        GLOBAL_SEARCH_FILTERS.BRANCHES,
        GLOBAL_SEARCH_FILTERS.AUTHORS,
        GLOBAL_SEARCH_FILTERS.COMMITTERS,
    ],
    ARTIFACT_TAGS: [
        GLOBAL_SEARCH_FILTERS.REPOS,
        GLOBAL_SEARCH_FILTERS.BRANCHES,
        GLOBAL_SEARCH_FILTERS.AUTHORS,
    ],
    APPLICATIONS: [
        GLOBAL_SEARCH_FILTERS.APPLICATION_STATUSES,
    ],
    DEPLOYMENTS: [
        GLOBAL_SEARCH_FILTERS.APPLICATION_STATUSES,
        GLOBAL_SEARCH_FILTERS.APP_NAME,
    ],
    TEMPLATES: [
        GLOBAL_SEARCH_FILTERS.REPOS,
        GLOBAL_SEARCH_FILTERS.BRANCHES,
    ]
};


@Component({
    selector: 'ax-global-search-filter',
    templateUrl: './global-search-filter.html',
    styles: [require('./global-search-filter.scss')],
})
export class GlobalSearchFilterComponent implements OnChanges, OnDestroy {
    public statusesFilter: FilterMultiSelect = {
        items: [
            {name: TASK_STATUSES.SUCCESSFUL, value: TASK_STATUSES.SUCCESSFUL, checked: false},
            {name: TASK_STATUSES.FAILED, value: TASK_STATUSES.FAILED, checked: false},
            {name: TASK_STATUSES.IN_PROGRESS, value: TASK_STATUSES.IN_PROGRESS, checked: false},
            {name: TASK_STATUSES.QUEUED, value: TASK_STATUSES.QUEUED, checked: false},
            {name: TASK_STATUSES.CANCELLED, value: TASK_STATUSES.CANCELLED, checked: false},
        ],
        messages: {
            name: 'Status'
        },
        isVisible: false,
        isStaticList: true
    };

    public applicationStatusFilter: FilterMultiSelect = {
        items: [
            {name: APPLICATION_STATUSES.ACTIVE, value: APPLICATION_STATUSES.ACTIVE, checked: false},
            {name: APPLICATION_STATUSES.ERROR, value: APPLICATION_STATUSES.ERROR, checked: false},
            {name: APPLICATION_STATUSES.INIT, value: APPLICATION_STATUSES.INIT, checked: false},
            {name: APPLICATION_STATUSES.STOPPED, value: APPLICATION_STATUSES.STOPPED, checked: false},
            {name: APPLICATION_STATUSES.STOPPING, value: APPLICATION_STATUSES.STOPPING, checked: false},
            {name: APPLICATION_STATUSES.TERMINATED, value: APPLICATION_STATUSES.TERMINATED, checked: false},
            {name: APPLICATION_STATUSES.TERMINATING, value: APPLICATION_STATUSES.TERMINATING, checked: false},
            {name: APPLICATION_STATUSES.WAITING, value: APPLICATION_STATUSES.WAITING, checked: false},
        ],
        messages: {
            name: 'Status',
        },
        isVisible: false,
        isStaticList: true
    };

    public authorFilter: FilterMultiSelect = {
        items: [],
        messages: {
            name: 'Author',
            emptyInput: 'Enter author name',
            notEmptyInput: 'Continue typing to refine further'
        },
        isVisible: false
    };

    public committerFilter: FilterMultiSelect = {
        items: [],
        messages: {
            name: 'Committer',
            emptyInput: 'Enter committer name',
            notEmptyInput: 'Continue typing to refine further'
        },
        isVisible: false
    };

    public artifactTagFilter: FilterMultiSelect = {
        items: [],
        messages: {
            name: 'Artifact Tag',
            emptyInput: 'Enter artifact tag',
            notEmptyInput: 'Continue typing to refine further'
        },
        isVisible: false
    };

    public branchFilter: FilterMultiSelect = {
        items: [],
        messages: {
            name: 'Branch',
            emptyInput: 'Enter branch name',
            notEmptyInput: 'Continue typing to refine further'
        },
        isVisible: false
    };

    public repoFilter: FilterMultiSelect = {
        items: [],
        messages: {
            name: 'Repository',
            emptyInput: 'Enter repo name',
            notEmptyInput: 'Continue typing to refine further'
        },
        isVisible: false,
        isStaticList: true
    };

    public appNameFilter: FilterMultiSelect = {
        items: [],
        messages: {
            name: 'Application name',
            emptyInput: 'Enter application name',
            notEmptyInput: 'Continue typing to refine further'
        },
        isVisible: false,
        isStaticList: true
    };

    @Input()
    public filters: GlobalSearchFilters;

    @Input()
    public category: string;

    @Output()
    public onFilterChange: EventEmitter<any> = new EventEmitter();

    private branchSubscription: Subscription;
    private committerSubscription: Subscription;
    private appNameSubscription: Subscription;
    private authorSubscription: Subscription;

    constructor(private usersService: UsersService,
                private artifactsService: ArtifactsService,
                private branchServices: BranchService,
                private applicationsService: ApplicationsService,
                private repoService: RepoService) {
    }

    public ngOnChanges() {
        this.statusesFilter.isVisible = this.isFilterVissible(GLOBAL_SEARCH_FILTERS.STATUSES);
        this.authorFilter.isVisible = this.isFilterVissible(GLOBAL_SEARCH_FILTERS.AUTHORS);
        this.committerFilter.isVisible = this.isFilterVissible(GLOBAL_SEARCH_FILTERS.COMMITTERS);
        this.repoFilter.isVisible = this.isFilterVissible(GLOBAL_SEARCH_FILTERS.REPOS);
        this.branchFilter.isVisible = this.isFilterVissible(GLOBAL_SEARCH_FILTERS.BRANCHES);
        this.applicationStatusFilter.isVisible = this.isFilterVissible(GLOBAL_SEARCH_FILTERS.APPLICATION_STATUSES);
        this.artifactTagFilter.isVisible = this.isFilterVissible(GLOBAL_SEARCH_FILTERS.ARTIFACT_TAGS);
        this.appNameFilter.isVisible = this.isFilterVissible(GLOBAL_SEARCH_FILTERS.APP_NAME);

        // select static elements after reload
        // STATUSES
        if (this.statusesFilter.isVisible) {
            this.statusesFilter.items.map(item => {
                item.checked = this.filters[this.category].statuses.indexOf(item.value) !== -1;
                return item;
            });
        }

        // APPLICATION STATUSESS
        if (this.applicationStatusFilter.isVisible) {
            this.applicationStatusFilter.items.map(item => {
                item.checked = this.filters[this.category].application_statuses.indexOf(item.value) !== -1;
                return item;
            });
        }
    }

    public ngOnDestroy() {
        if (this.branchSubscription) {
            this.branchSubscription.unsubscribe();
        }
        if (this.committerSubscription) {
            this.committerSubscription.unsubscribe();
        }
        if (this.authorSubscription) {
            this.authorSubscription.unsubscribe();
        }
        if (this.appNameSubscription) {
            this.appNameSubscription.unsubscribe();
        }
    }

    public onStatusChange(statuses: string[]) {
        this.filters[this.category].statuses = statuses;

        this.onFilterChange.emit(this.filters);
    }

    public onApplicationStatusChange(statuses: string[]) {
        this.filters[this.category].application_statuses = statuses;

        this.onFilterChange.emit(this.filters);
    }

    public onAuthorChange(authors: string[]) {
        this.filters[this.category].authors = authors;

        this.onFilterChange.emit(this.filters);
    }

    public onAuthorQuery(searchString: string) {
        if (this.authorSubscription) {
            this.authorSubscription.unsubscribe();
        }

        let params = {
            search: searchString,
        };

        this.authorSubscription = this.usersService.getUsers(params, true).subscribe(res => {
            this.authorFilter.items = res.data.map(user => {
                return {name: user.username, value: user.username, checked: false};
            });
        }, error => {
            this.authorFilter.items = [];
        });
    }

    public onCommitterChange(authors: string[]) {
        this.filters[this.category].committers = authors;

        this.onFilterChange.emit(this.filters);
    }

    public onCommitterQuery(searchString: string) {
        if (this.committerSubscription) {
            this.committerSubscription.unsubscribe();
        }

        let params = {
            search: searchString,
        };

        this.committerSubscription = this.usersService.getUsers(params, true).subscribe(res => {
            this.committerFilter.items = res.data.map(user => {
                return {name: user.username, value: user.username, checked: false};
            });
        }, error => {
            this.committerFilter.items = [];
        });
    }

    public onApplicationNameChange(appNames: string[]) {
        this.filters[this.category].app_name = appNames;

        this.onFilterChange.emit(this.filters);
    }

    public onApplicationNameQuery(searchString: string) {
        if (this.appNameSubscription) {
            this.appNameSubscription.unsubscribe();
        }

        let params = {
            search: searchString,
            limit: 100,
            searchFields: ['name'],
            sort: 'name',
        };

        this.appNameSubscription = this.applicationsService.getApplications(params, true).subscribe(res => {
            this.appNameFilter.items = res.map(app => {
                return {name: app.name, value: app.name, checked: false};
            });
        }, error => {
            this.appNameFilter.items = [];
        });
    }

    public onArtifactTagChange(artifactTags: string[]) {
        this.filters[this.category].artifact_tags = artifactTags;

        this.onFilterChange.emit(this.filters);
    }

    public onArtifactTagQuery(searchString: string) {
        let params = {
            search: searchString,
        };

        this.artifactsService.getArtifactTags(params, true).subscribe(res => {
            this.artifactTagFilter.items = res.data.map(tag => {
                return {name: tag, value: tag, checked: false};
            });
        }, error => {
            this.artifactTagFilter.items = [];
        });
    }

    public onBranchChange(branches: {repo: string, branch: string}[]) {
        this.filters[this.category].branch = branches;

        this.onFilterChange.emit(this.filters);
    }

    public onBranchQuery(searchString: string) {
        if (this.branchSubscription) {
            this.branchSubscription.unsubscribe();
        }

        let params = {
            name: searchString,
            limit: 100
        };

        this.branchSubscription = this.branchServices.getBranchesAsync(params, true).subscribe(res => {
            this.branchFilter.items = res.data.map(branch => {
                return {name: branch.name, value: `${branch.repo} ${branch.name}`, checked: false, subname: branch.repo};
            });
        }, error => {
            this.branchFilter.items = [];
        });
    }

    public onRepoChange(repos: string[]) {
        this.filters[this.category].repo = repos;

        this.onFilterChange.emit(this.filters);
    }

    public onRepoQuery() {
        if (this.repoFilter.isVisible && this.repoFilter.items.length === 0) {
            this.repoService.getReposAsync(true).subscribe(res => {
                this.repoFilter.items = res.data.map(repo => {
                    return {name: repo.substring(repo.lastIndexOf('/') + 1), value: repo, checked: false, subname: repo};
                });
            }, error => {
                this.repoFilter.items = [];
            });
        }
    }

    private isFilterVissible(filterName: string): boolean {
        return GLOBAL_SEARCH_FILTER_CONFIG[this.category.toUpperCase()].indexOf(filterName) !== -1;
    }
}

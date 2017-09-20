import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { BehaviorSubject } from 'rxjs/BehaviorSubject';

import { Project, PROMOTED_CATEGORY_NAME } from '../../../model';
import { ProjectService } from '../../../services';
import { LayoutSettings, HasLayoutSettings } from '../../layout';
import { ViewUtils, GlobalSearchSetting, LOCAL_SEARCH_CATEGORIES, BranchesFiltersComponent } from '../../../common';

@Component({
    selector: 'ax-catalog-overview',
    templateUrl: './ax-catalog-overview.html',
    styles: [ require('./ax-catalog-overview.scss') ],
})
export class AxCatalogOverviewComponent implements OnInit, HasLayoutSettings, LayoutSettings {

    public allProjects: Project[] = [];
    public selectedProject: Project;
    public categories: {name: string; projects: Project[] }[] = [];
    public selectedCategory: string;
    public branch: string;
    public repo: string;
    public searchString: string;

    constructor(
        private router: Router,
        private route: ActivatedRoute,
        private projectService: ProjectService) {
    }

    public ngOnInit() {
        this.route.params.subscribe(params => {
            this.searchString = params['searchString'];
            this.branch = params['branch'] ? decodeURIComponent(params['branch']) : null;
            this.repo = params['repo'] ? decodeURIComponent(params['repo']) : null;

            if (this.searchString) {
                this.globalSearch.value.searchString = this.searchString;
                this.globalSearch.value.keepOpen = true;

                this.projectService.getProjects({
                    search: this.searchString,
                    repo: this.repo,
                    branch: this.branch,
                    published: this.branch && this.repo ? null : true,
                }).then(projects => this.allProjects = projects);
            } else {
                this.projectService.getProjectsByCategory({
                    repo: this.repo,
                    branch: this.branch,
                    published: this.branch && this.repo ? null : true,
                }).then(categoryToProject => {
                    this.categories = Array.from(categoryToProject.entries())
                        .filter(([category]) => category !== PROMOTED_CATEGORY_NAME)
                        .map(([category, projects]) => {
                            return { name: category, projects };
                        });
                    this.selectedCategory = this.categories.length > 0 && this.categories[0].name;
                });
            }
        });
    }

    get layoutSettings(): LayoutSettings {
        return {
            pageTitle: 'Catalog',
            branchNavPanelUrl: '/app/ax-catalog',
            breadcrumb: this.breadcrumb,
            globalSearch: this.globalSearch,
            hasTabs: true,
        };
    }

    public globalSearch: BehaviorSubject<GlobalSearchSetting> = new BehaviorSubject<GlobalSearchSetting>({
        suppressBackRoute: false,
        keepOpen: false,
        searchCategory: LOCAL_SEARCH_CATEGORIES.CATALOG.name,
        searchString: this.searchString,
        applyLocalSearchQuery: (searchString) => {
            this.searchString = searchString;
            let params = ViewUtils.sanitizeRouteParams(this.getRouteParams(), { searchString });

            this.router.navigate(['/app/ax-catalog', params], {relativeTo: this.route});
        },
    });

    get breadcrumb(): { title: string, routerLink: any[] }[] {
        let arr: any[] = [{
            title: 'All Apps',
            routerLink: this.repo ? [`/app/ax-catalog`] : null
        }];

        if (this.repo) {
            arr.push({
                title: BranchesFiltersComponent.parseRepoUrl(this.repo).name,
            });
        }

        if (this.branch) {
            arr.push({
                title: this.branch,
            });
        }

        return arr;
    }

    private getRouteParams() {
        let params = {};

        if (this.searchString) {
            params['searchString'] = encodeURIComponent(this.searchString);
        }

        if (this.branch) {
            params['branch'] = encodeURIComponent(this.branch);
        }

        if (this.repo) {
            params['repo'] = encodeURIComponent(this.repo);
        }

        return ViewUtils.sanitizeRouteParams(params);
    }
}

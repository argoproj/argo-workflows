import { Component, Input, OnChanges, OnDestroy } from '@angular/core';
import { Subscription } from 'rxjs';

import { Pagination, InitFilters } from '../../../common';
import { TemplateFieldNames, Template } from '../../../model';
import { TemplateService, GlobalSearchService } from '../../../services';

@Component({
    selector: 'ax-templates-list',
    templateUrl: './templates-list.html',
    styles: [require('./templates-list.scss')],
})
export class TemplatesListComponent implements OnChanges, OnDestroy {
    protected readonly limit: number = 10;

    @Input()
    public filters: InitFilters;

    @Input()
    public searchString: string;

    public templates: Template[] = [];
    public params: InitFilters;
    public dataLoaded: boolean = false;
    public pagination: Pagination = {
        limit: this.limit,
        offset: 0,
        listLength: this.templates.length
    };

    private subscriptions: Subscription[] = [];

    constructor(private templateService: TemplateService,
                private globalSearchService: GlobalSearchService) {
    }

    public ngOnChanges() {
        this.params = {
            branch: this.filters.branch,
            repo: this.filters.repo
        };
        // restart pagination if changed search parameters
        this.pagination = { limit: this.limit, offset: 0, listLength: this.templates.length };
        this.updateTemplates( this.params, this.pagination, false);
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
    }

    public onPaginationChange(pagination: Pagination) {
        this.updateTemplates(this.params, { offset: pagination.offset, limit: pagination.limit }, false);
    }

    public navigateToDetails(id: string): void {
        this.globalSearchService.navigate(['/app/service-catalog/history/', id]);
    }

    private updateTemplates(params: InitFilters, pagination: Pagination, hideLoader?: boolean) {
        this.dataLoaded = false;
        pagination.limit += 1;

        this.subscriptions.push(this.getTemplates(params, pagination, hideLoader).subscribe(result => {
            this.dataLoaded = true;

            this.templates = result.data.slice(0, this.limit);

            this.pagination = {
                offset: pagination.offset,
                limit: this.limit,
                listLength: this.templates.length,
                hasMore: result.data.length > this.limit
            };
        }, error => {
            this.dataLoaded = true;
            this.templates = [];
        }));
    }

    private getTemplates(params: InitFilters, pagination: Pagination, hideLoader?: boolean) {
        let parameters = {
            status: null,
            tags: null,
            limit: null,
            offset: null,
            repo: null,
            branches: null,
            username: null,
            fields: [
                TemplateFieldNames.name,
                TemplateFieldNames.description,
                TemplateFieldNames.repo,
                TemplateFieldNames.branch,
                TemplateFieldNames.type,
            ],
            searchFields: [
                TemplateFieldNames.name,
                TemplateFieldNames.description,
                TemplateFieldNames.type,
                TemplateFieldNames.repo,
                TemplateFieldNames.branch,
            ],
            search: this.searchString
        };

        if (pagination.offset) {
            parameters.offset = pagination.offset;
        }

        if (pagination.limit) {
            parameters.limit = pagination.limit;
        }

        if (params.repo && params.repo.length) {
            parameters.repo = params.repo;
        }

        if (params.branch && params.branch.length) {
            parameters.branches = params.branch.map(i => {
                return { repo: i.split(' ')[0], name: i.split(' ')[1] };
            });
        }

        return this.templateService.getTemplatesAsync(parameters, hideLoader);
    }
}

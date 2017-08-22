import { Component, OnInit, Input, OnDestroy, OnChanges } from '@angular/core';

import {TemplateService} from '../../services';
import {Template} from '../../model';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';
import { Subscription, Observable } from 'rxjs';
import * as _ from 'lodash';

@Component({
    selector: 'ax-templates-filters',
    templateUrl: './templates-filters.html',
    styles: [ require('./templates-filters.scss') ],
})
export class TemplatesFiltersComponent implements OnInit, OnDestroy, OnChanges {

    @Input()
    public selectedRepo: string;

    @Input()
    public selectedBranch: string;

    @Input() set selectedTemplateIds(value: string) {
        if (value) {
            this._selectedTemplateIds = value;
        } else {
            this._selectedTemplateIds = null;
            this.selectedTemplates = [];
            this.templates.forEach((t: Template) => t.selected = false);
            this.selectedTemplateNames.next(null);
        }
    }

    public selectedTemplateNames: BehaviorSubject<string> = new BehaviorSubject(null);
    public templates: Template[] = [];
    public loading: boolean = false;
    private selectedTemplates: Template[] = [];
    private canScroll: boolean = false;
    private subscriptions: Subscription[] = [];
    private _selectedTemplateIds: string = null;

    constructor(private templateService: TemplateService) {
    }

    public ngOnInit() {
        this.updateSelectedValues();
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
    }

    // if repo/branch changed, force to get new templates
    public ngOnChanges() {
        this.templates = [];
        this.canScroll = true;
    }

    public onScroll() {
        if (this.canScroll) {
            this.loadTemplates(this.templates.length);
        }
    }

    public loadTemplates(offset: number) {
        this.canScroll = false;
        this.loading = true;
        let pageSize = 20;
        this.subscriptions.push(this.templateService.getTemplatesAsync({
            fields: ['id', 'name', 'repo', 'branch'],
            sort: 'name',
            repo: this.selectedRepo ? this.selectedRepo : null,
            branch: this.selectedBranch ? this.selectedBranch : null,
            limit: pageSize,
            offset: offset,
        }, false).subscribe(success => {
            this.templates = this.templates.concat(success.data || []);
            this.canScroll = (success.data || []).length >= pageSize;
            this.loading = false;
            this.updateSelectedValues();
        }));
    }

    get selectedTemplateIds(): string {
        return this.selectedTemplates.map(t => t.id).join(',');
    }

    selectTemplate(template) {
        template.selected = !template.selected;
        if (template.selected) {
            this.selectedTemplates.push(template);
        } else {
            _.remove(this.selectedTemplates, (t: Template) => t.id === template.id);
        }
        this.selectedTemplateNames.next(this.selectedTemplates.map(t => t.name).join(','));
    }

    private updateSelectedValues() {
        if (this._selectedTemplateIds) {
            let observableList: Observable<any>[] = [];
            this._selectedTemplateIds.split(',').forEach((selectedId: string) => {
                let template = this.templates.find((t: Template) => t.id === selectedId);
                if (template) {
                    template.selected = true;
                    if (!this.selectedTemplates.find((t: Template) => t.id === template.id)) {
                        this.selectedTemplates.push(template);
                    }
                } else {
                    observableList.push(this.templateService.getTemplateByIdAsync(selectedId));
                }
            });
            this.subscriptions.push(Observable.forkJoin(observableList).subscribe((success: any) => {
                success.forEach((template: Template) => {
                    if (!this.selectedTemplates.find((t: Template) => t.id === template.id)) {
                        this.selectedTemplates.push(template);
                        this.selectedTemplateNames.next(this.selectedTemplates.map(t => t.name).join(','));
                    }
                });
            }));
            this.selectedTemplateNames.next(this.selectedTemplates.map(t => t.name).join(','));
        }
    }
}

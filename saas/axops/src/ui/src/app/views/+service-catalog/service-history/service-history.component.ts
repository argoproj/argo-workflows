import { Component, OnInit, OnDestroy, AfterViewInit, ViewChild } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';

import { TaskService, TemplateService, GlobalSearchService, ViewPreferencesService } from '../../../services';
import { Task, Template, ViewPreferences } from '../../../model';
import { LayoutSettings } from '../../layout';
import { LaunchPanelService, ViewUtils } from '../../../common';
import { TemplateViewerComponent } from '../../../common/template-viewer/template-viewer.component';

@Component({
    selector: 'ax-service-history',
    templateUrl: './service-history.html',
})
export class ServiceHistoryComponent implements LayoutSettings, OnInit, AfterViewInit, OnDestroy {
    public canLoadMore: boolean = false;
    public isEmptyList: boolean = true;

    private tasks: Task[] = [];
    private templateId: string;
    private isLoaded: boolean = false;
    private template: Template;
    private offset: number = 0;
    private readonly limit: number = 20;
    private subscription: Subscription;
    private backToSearchUrl: string;
    private subscriptions: Subscription[] = [];
    private viewPreferences: ViewPreferences;

    @ViewChild('templateViewer')
    public templateViewer: TemplateViewerComponent;

    constructor(
        private router: Router,
        private taskService: TaskService,
        private templateService: TemplateService,
        private route: ActivatedRoute,
        private launchPanelService: LaunchPanelService,
        private globalSearchService: GlobalSearchService,
        private viewPreferencesService: ViewPreferencesService) {
    }

    public branchNavPanelUrl = '/app/service-catalog/overview';

    public ngOnInit() {
        this.route.params.subscribe(params => {
            this.templateId = params['id'];
            this.getJobsHistory();
            this.getTemplateById(this.templateId);
        });
        this.subscriptions.push(this.viewPreferencesService.getViewPreferencesObservable().subscribe(viewPreferences => this.viewPreferences = viewPreferences));
    }

    public ngAfterViewInit() {
        this.backToSearchUrl = this.globalSearchService.popBackToSearchUrl();
    }

    public ngOnDestroy() {
        if (this.subscription) {
            this.subscription.unsubscribe();
        }
        this.subscriptions.forEach(item => item.unsubscribe());
        this.subscriptions = [];
    }

    public onBackToSearch() {
        this.router.navigateByUrl(this.backToSearchUrl);
    }

    get isYamlVisible() {
        return this.templateViewer && this.templateViewer.isYamlVisible;
    }

    get templateName(): string {
        return this.template && this.template.name;
    }

    get pageTitle(): string {
        return this.templateName ? this.templateName : '';
    }

    get breadcrumb(): { title: string, routerLink: any[] }[] {
        return this.template ?
            ViewUtils.getBranchBreadcrumb(this.template.repo, this.template.branch, '/app/service-catalog/overview', this.viewPreferences, this.template.name) : null;
    }

    get hasTabs(): boolean {
        return true;
    }

    public globalAddAction() {
        this.createNewJob();
    }

    public createNewJob() {
        this.launchPanelService.openPanel(null, this.template, false);
    }

    public openJob(jobId: string) {
        this.router.navigate(['/app/timeline/jobs/', jobId]);
    }

    public onLoadMore() {
        if (this.canLoadMore) {
            this.canLoadMore = false;
            this.getJobsHistory();
        }
    }

    public selectedTab() {
        this.templateViewer.closeYaml();
    }

    private getJobsHistory() {
        this.isLoaded = false;
        this.subscription = this.taskService.getTasks(
            {
                templateIds: this.templateId,
                limit: this.limit,
                offset: this.offset,
                fields: ['id', 'name', 'commit', 'cost'],
                isActive: false,
            }, true)
            .subscribe(
                success => {
                    this.offset += this.limit;
                    this.canLoadMore = success.data.length === this.limit;
                    this.tasks = this.tasks.concat(success.data);
                    this.isLoaded = true;
                    this.isEmptyList = this.tasks.length === 0;
                }
            );
    }

    private getTemplateById(templateId: string) {
        this.templateService.getTemplateByIdAsync(templateId).subscribe(template => {
            this.template = template;
        });
    }
}

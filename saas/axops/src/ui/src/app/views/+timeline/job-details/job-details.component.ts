import * as moment from 'moment';
import { Component, OnInit, OnDestroy, NgZone, ViewChild, AfterViewInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Observable, Subscription } from 'rxjs';

import { LogsComponent, MenuItem, DropdownMenuSettings, NotificationsService, Tab } from 'argo-ui-lib/src/components';

import { Task, TaskStatus, Commit, Template, ViewPreferences } from '../../../model';
import { ArtifactsComponent, ViewUtils } from '../../../common';
import { LayoutSettings, HasLayoutSettings } from '../../layout';
import {
    TaskService, FixtureService, GlobalSearchService, AuthenticationService, ViewPreferencesService, ToolService, JiraService, TemplateService
} from '../../../services';

import { NodeInfo, JobTreeNode } from '../../../common/workflow-tree/workflow-tree.view-models';
import { StepInfo } from '../jobs.view-models';
import { JobsService } from '../jobs.service';
import { RecentCommitsComponent } from '../recent-commits/recent-commits.component';
import { JobsHistoryComponent } from '../jobs-history/jobs-history.component';


const REFRESH_INTERVAL = 1000;

@Component({
    selector: 'ax-job-details',
    templateUrl: './job-details.html',
    styles: [ require('./job-details.scss') ],
})
export class JobDetailsComponent implements OnInit, AfterViewInit, OnDestroy, LayoutSettings, HasLayoutSettings {
    public task: Task = new Task();
    public taskOriginalTemplate: Template;
    public selectedTabKey: string;
    public flatListOfSteps: StepInfo[] = [];
    public stepsExpanded: boolean = true;
    public selectedNode: NodeInfo;
    public selectedNodeDetailsTab: string;
    public selectedNodeMenuItems: MenuItem[];
    public taskMenuItems: MenuItem[];
    public selectedFixture: Task;
    public fixtureStatuses = TaskStatus;
    public tailLogs: boolean = true;
    public isYamlVisible: boolean = false;
    public isRecentCommitsVisible: boolean = false;
    public isJobHistoryVisible: boolean = false;
    public isJiraCreatePanelVisible: boolean = false;
    public isJiraIssueListVisible: boolean = false;
    public isArtifactTagManagementPanelVisible: boolean;
    public artifactTags: string[];
    public selectedStepName: string;
    public artifactDownloadMenu: DropdownMenuSettings;
    public isJiraConfigured: boolean;
    public actionMenu: MenuItem[];
    public hasTabs: boolean = true;

    private _selectedStepId: string;
    private stepNameToInfo: Map<string, StepInfo> = new Map<string, StepInfo>();
    private browseStepArtifact: string;
    private consoleStep: string;
    private subscriptions: Subscription[] = [];
    private eventsSubscription: Subscription = null;
    private timeRefreshSubscription: Subscription;
    private selectedArtifactsCount: number = 0;
    private backToSearchUrl: string;
    private isCurrentUserAuthenticated: boolean;
    private viewPreferences: ViewPreferences;

    // View related property
    private taskLoaded = false;

    @ViewChild(RecentCommitsComponent)
    private recentCommitsComponent: RecentCommitsComponent;

    @ViewChild(LogsComponent)
    private jobLogsComponent: LogsComponent;

    @ViewChild(JobsHistoryComponent)
    private jobsHistoryComponent: JobsHistoryComponent;

    @ViewChild(ArtifactsComponent)
    private artifactsComponent: ArtifactsComponent;

    constructor(private router: Router,
                private taskService: TaskService,
                private route: ActivatedRoute,
                private jobsService: JobsService,
                private fixtureService: FixtureService,
                private notificationsService: NotificationsService,
                private zone: NgZone,
                private toolService: ToolService,
                private jiraService: JiraService,
                private globalSearchService: GlobalSearchService,
                private authenticationService: AuthenticationService,
                private viewPreferencesService: ViewPreferencesService,
                private templateService: TemplateService) {

        this.authenticationService.getCurrentUser().then(user => {
            this.isCurrentUserAuthenticated = !user.anonymous;
        });
    }

    get layoutSettings(): LayoutSettings {
        return this;
    }

    get pageTitle(): string {
        return this.task.name;
    }

    public branchNavPanelUrl = '/app/timeline';

    get globalAddActionMenu(): DropdownMenuSettings {
        let items = this.taskMenuItems || [];
        if (this.isJiraConfigured) {
            items = items.concat([{
                title: 'Create JIRA Issue',
                iconName: 'ax-icon-jira',
                action: () => this.onViewJiraCreateIssue(),
            }]);
        }
        return new DropdownMenuSettings(this.isCurrentUserAuthenticated ? items : [], 'fa-ellipsis-v');
    }

    ngOnInit() {
        this.subscriptions.push(this.route.params.subscribe(params => {
            this.selectedTabKey = params['tab'] || 'workflow';
            this.selectedStepName = params['step'];
            this._selectedStepId = params['step_id'];
            this.browseStepArtifact = params['browseStepArtifact'] ? decodeURIComponent(params['browseStepArtifact']) : null;
            this.consoleStep = params['consoleStep'];
            this.selectedNode = (this.consoleStep || this.browseStepArtifact) ? null : this.selectedNode;
            this.isYamlVisible = params['isYamlVisible'] === 'true';
            this.isRecentCommitsVisible = params['isRecentCommitsVisible'] === 'true';
            this.isJobHistoryVisible = params['isJobHistoryVisible'] === 'true';
            this.isJiraCreatePanelVisible = params['isJiraCreatePanelVisible'] === 'true';
            this.isJiraIssueListVisible = params['isJiraIssueListVisible'] === 'true';
            this.isArtifactTagManagementPanelVisible = params['isArtifactTagManagementVisible'] === 'true';
            if (!this.task || this.task.id !== params['id']) {
                this.getDetails(params['id']);
            } else {
                this.showSelectedPanel();
            }
        }));

        this.subscriptions.push(this.jiraService.showJiraIssueCreatorPanel.subscribe(res => {
            if (!res.isVisible && !this.isJiraIssueListVisible) {
                this.closeSlidingPanel();
            }
        }));

        this.subscriptions.push(this.jiraService.showJiraIssuesListPanel.subscribe(res => {
            if (!res.isVisible) {
                this.closeSlidingPanel();
            }
        }));

        this.subscriptions.push(this.jiraService.jiraIssueCreated.subscribe(res => {
            this.getDetails(this.task.id);
            this.onViewJiraIssuesList();
        }));

        this.subscriptions.push(this.viewPreferencesService.getViewPreferencesObservable().subscribe(viewPreferences => this.viewPreferences = viewPreferences));

        this.subscriptions.push(this.toolService.isJiraConfigured().subscribe(isConfigured => {
            this.isJiraConfigured = isConfigured;
        }));
    }

    ngAfterViewInit() {
        this.backToSearchUrl = this.globalSearchService.popBackToSearchUrl();
    }

    ngOnDestroy() {
        this.ensureTimeRefreshUnsubscribed();
        this.ensureEventsUnsubscribed();
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
    }

    getLogsSource(task: Task, isFixture?: boolean) {
        return {
            loadLogs: () => {
                if (isFixture ||
                    [TaskStatus.Running, TaskStatus.Success, TaskStatus.Failed, TaskStatus.Cancelled].indexOf(task.status) > -1) {
                    return this.taskService.getTaskLogs(task.id);
                }
                return null;
            },
            getKey() {
                return `${task.id}_${task.status}_${isFixture}`;
            }
        };
    }

    onSelectedArtifactCountChanged(count: number) {
        this.selectedArtifactsCount = count;
    }

    onDownloadArtifacts() {
        this.artifactsComponent.downloadSelectedArtifacts();
    }

    onBackToSearch() {
        this.router.navigateByUrl(this.backToSearchUrl);
    }

    get selectedStepId(): string {
        if ((!this._selectedStepId || this._selectedStepId === '') && this.flatListOfSteps.length > 0) {
            return this.flatListOfSteps[0].value.id;
        }
        return this._selectedStepId;
    }

    set selectedStepId(value: string) {
        this._selectedStepId = value;
    }

    getStep(id): StepInfo {
        return this.stepNameToInfo.get(id);
    }

    get getConsoleStepId(): string {
        return this.consoleStep;
    }

    get selectedTask(): Task {
        let step = this.getStep(this.selectedStepId);
        return step ? step.value : null;
    }

    /**
     * Merges calculated run time from existing tasks to avoid task duration jumping on workflow tab.
     */
    mergeTasks(task: Task, newTask: Task) {
        let idToRunTime = new Map<string, number>();
        if (task.children) {
            task.children.forEach(item => idToRunTime.set(item.id, item.run_time));
        }
        if (newTask.children) {
            newTask.children.forEach(item => item.run_time = idToRunTime.get(item.id));
        }
        return newTask;
    }

    /**
     * Load the task details from service api
     */
    getDetails(serviceId, autoRefresh = false, hideLoader = false) {
        this.subscriptions.push(this.taskService.getTask(serviceId, autoRefresh || hideLoader).subscribe(
            success => {
                this.task = autoRefresh && this.task ? this.mergeTasks(this.task, success) : success;
                if (!this.taskOriginalTemplate || this.taskOriginalTemplate.id !== this.task.template_id) {
                    // try to load template and fallback to task template if original has been deleted
                    this.templateService.getTemplateByIdAsync(this.task.template_id, true).subscribe(
                        res => this.taskOriginalTemplate = res, err => this.taskOriginalTemplate = this.task.template, () => {
                            if (this.isJobHistoryVisible) {
                                this.jobsHistoryComponent.template = this.taskOriginalTemplate;
                                this.jobsHistoryComponent.loadJobsHistory();
                            }
                        });
                }
                this.artifactTags = this.task.artifact_tags !== '' ? JSON.parse(this.task.artifact_tags) : [];
                this.taskMenuItems = [{
                    title: 'Artifact tags',
                    iconName: 'ax-icon-artifact',
                    action: () => this.toggleArtifactTagManagementPanel(true)
                }].concat(this.jobsService.getJobMenu(this.task).menu);

                // Tell the page that task is loaded - helps in reducing some computations
                this.taskLoaded = true;
                if (!autoRefresh) {
                    this.subscribeToEvents(serviceId);
                }
                if (this.task.status === TaskStatus.Running && !this.timeRefreshSubscription) {
                    this.timeRefreshSubscription = Observable.interval(REFRESH_INTERVAL).subscribe(() => {
                        this.task.children.forEach(step => {
                            let now = moment().unix();
                            if (step.status === TaskStatus.Running) {
                                step.run_time = now - step.launch_time - step.wait_time;
                            }
                        });
                    });
                } else if (this.task.status !== TaskStatus.Running) {
                    this.ensureTimeRefreshUnsubscribed();
                }
                this.flatListOfSteps = this.calculateBoundaries(
                    this.getFlatListOfSteps(this.mapTaskToStep(this.task).children, null)
                ).reverse();
                this.stepNameToInfo.clear();
                this.flatListOfSteps.forEach(step => this.stepNameToInfo.set(step.value.id, step));

                // if it's needed, get data after reload
                this.showSelectedPanel();
            }
        ));
    }

    tabChange(selectedTab: Tab) {
        this.selectedArtifactsCount = 0;
        this.router.navigate(['/app/timeline/jobs/', this.task.id, {
            tab: selectedTab.tabKey
        }]);
    }

    selectStep(stepId: string) {
        this.router.navigate(['/app/timeline/jobs/', this.task.id, {
            tab: this.selectedTabKey,
            step_id: stepId
        }]);
    }

    public selectFixture(fixture: Task) {
        this.selectedFixture = fixture;
    }

    trackByCommitRevision(commit: Commit) {
        return commit.revision;
    }

    trackByTaskId(task: Task) {
        return task.id;
    }

    trackByStepName(step: StepInfo) {
        return step.name;
    }

    closeSlidingPanel() {
        this.selectedNode = null;
        this.selectedFixture = null;
        this.selectedNodeMenuItems = null;
        let params = <any>{};
        if (this.selectedTabKey) {
            params.tab = this.selectedTabKey;
        }
        if (this.selectedStepId) {
            params.step_id = this.selectedStepId;
        }
        this.router.navigate(['/app/timeline/jobs/', this.task.id, params]);
    }

    closeJobsHistoryPanel() {
        this.jobsHistoryComponent.clearJobsHistory();
        this.closeSlidingPanel();
    }

    closeRecentCommitsPanel() {
        this.recentCommitsComponent.clearRecentCommits();
        this.closeSlidingPanel();
    }

    toggleTail() {
        this.tailLogs = !this.tailLogs;
    }

    get breadcrumb(): { title: string, routerLink: any[] }[] {
        return this.task ? ViewUtils.getBranchBreadcrumb(
            this.task.repo, this.task.branch, {url: '/app/timeline', params: { view: 'job' }}, this.viewPreferences,  `Job ID: ${this.task.id}`) : null;
    }

    public showNodeDetails(node: NodeInfo, tab: string) {
        this.closeYamlSlidingPanel();

        this.selectedNode = node;
        this.selectedNodeDetailsTab = tab || 'logs';

        if (this.selectedNode.workflow.fixturesUsed && this.selectedNode.workflow.fixturesUsed.length > 0) {
            this.selectedFixture = this.selectedNode.workflow.fixturesUsed[0];
        }

        this.selectedNodeMenuItems = this.jobsService.getActionMenuSettings(this.selectedNode.workflow.value, this.task).menu;
        this.actionMenu = new DropdownMenuSettings(this.selectedNodeMenuItems).menu;

        let step = this.jobsService.getSelectedStep(this.selectedNode.workflow.value, this.task);
        this.artifactDownloadMenu = new DropdownMenuSettings(this.jobsService.getArtifactMenuItems(this.task, step));
    }

    public toggleTailLogs() {
        this.tailLogs = !this.tailLogs;
    }

    public getFixtureMenu(fixture: Task) {
        return new DropdownMenuSettings(this.jobsService.getArtifactMenuItems(this.task, new JobTreeNode(fixture, 'test', [], fixture)));
    }

    public onViewRecentCommits() {
        this.router.navigate(['/app/timeline/jobs/', this.task.id, {
            tab: this.selectedTabKey,
            isRecentCommitsVisible: true
        }]);
    }

    public onViewJobsHistory() {
        this.router.navigate(['/app/timeline/jobs/', this.task.id, {
            tab: this.selectedTabKey,
            isJobHistoryVisible: true
        }]);
    }

    public onViewJiraCreateIssue() {
        this.router.navigate(['/app/timeline/jobs/', this.task.id, {
            tab: this.selectedTabKey,
            isJiraCreatePanelVisible: true
        }]);
    }

    public onViewJiraIssuesList() {
        this.router.navigate(['/app/timeline/jobs/', this.task.id, {
            tab: this.selectedTabKey,
            isJiraIssueListVisible: true
        }]);
    }

    public showYaml(node: NodeInfo | MouseEvent) {
        let navExtras = {
            tab: this.selectedTabKey,
            isYamlVisible: true,
        };
        if (node.hasOwnProperty('name')) {
            this.selectedStepId = node['workflow'].value.id;
            navExtras['step'] = node['name']; // required to select proper part in YAML panel
            navExtras['step_id'] = node['workflow'].value.id; // required to select proper step on workflow tree
        }

        this.router.navigate(['/app/timeline/jobs/', this.task.id, navExtras]);
    }

    public closeYamlSlidingPanel() {
        this.router.navigate(['/app/timeline/jobs/', this.task.id, {
            tab: this.selectedTabKey
        }]);
    }

    public getProgressClasses(workflow: JobTreeNode) {
        let status = workflow.value.status === TaskStatus.Failed ? 'failed' : 'running';
        let percentage = this.getPercentage(workflow);
        return [
            'job-details__node-progress', `job-details__node-progress--${percentage.toFixed()}-${status}`
        ].join(' ');
    }

    public getPercentage(workflow: JobTreeNode): number {
        let percentage;
        if (workflow.value.status === TaskStatus.Init) {
            percentage = 0;
        } else if (workflow.value.status === TaskStatus.Running || workflow.value.status === TaskStatus.Waiting) {
            if (workflow.children.length === 0) {
                percentage = Math.min((workflow.value.run_time / workflow.value.average_runtime) * 100, 95);
            } else {
                let childrenCount = 0;
                let sum = 0;
                workflow.children.forEach(children => children.forEach(item => {
                    childrenCount++;
                    sum += this.getPercentage(item);
                }));
                percentage = sum / childrenCount;
            }
        } else {
            percentage = 100;
        }
        return percentage;
    }

    public toggleArtifactTagManagementPanel(isVisible: boolean) {
        let navExtras = {
            tab: this.selectedTabKey
        };
        if (isVisible) {
            navExtras['isArtifactTagManagementVisible'] = isVisible;
        }

        this.router.navigate(['/app/timeline/jobs/', this.task.id, navExtras]);
    }

    public updateWorkflowData(hideLoader?: boolean) {
        this.getDetails(this.task.id, hideLoader || false);
    }

    public downloadUserLogs(task: Task): void {
        window.location.href = `v1/artifacts?action=download&service_instance_id=${task.id}&retention_tags=user-log`;
    }

    public isDownloadLogsEnabled(task: Task) {
        return task && (task.status === TaskStatus.Failed || task.status === TaskStatus.Success);
    }

    // Scroll page back to top
    public scrollToTop() {
        if (this.jobLogsComponent) {
            this.jobLogsComponent.scrollToTop();
        }
    }

    private subscribeToEvents(serviceId: string) {
        this.ensureEventsUnsubscribed();
        if ([TaskStatus.Init, TaskStatus.Waiting, TaskStatus.Running, TaskStatus.Canceling].indexOf(this.task.status) > -1) {
            this.eventsSubscription = this.taskService.getTaskStepEvents(serviceId).subscribe(
                () => this.zone.run(() => {
                    this.getDetails(serviceId, true);
                }),
                // Resubscribe of event stream failed/completed but task is still running.
                () => {
                    this.getDetails(serviceId, false, true);
                },
                () => {
                    this.getDetails(serviceId, false, true);
                });
        }
    }

    private ensureEventsUnsubscribed() {
        if (this.eventsSubscription) {
            this.eventsSubscription.unsubscribe();
            this.eventsSubscription = null;
        }
    }

    private ensureTimeRefreshUnsubscribed() {
        if (this.timeRefreshSubscription) {
            this.timeRefreshSubscription.unsubscribe();
            this.timeRefreshSubscription = null;
        }
    }

    private mapTaskToStep(task: Task): StepInfo {
        let children: StepInfo[];
        if (task.template.type === 'container') {
            let step = {};
            step[task.name] = task;
            task.children = [task];
            children = this.mapStepsToSteps([step]);
        } else {
            children = task.template.hasOwnProperty('steps') ? this.mapStepsToSteps(task.template.steps) : [];
        }
        return {
            name: task.name,
            value: task,
            children: children,
            isSkipped: task.status === TaskStatus.Skipped,
            isSucceeded: task.status === TaskStatus.Success,
            isFailed: task.status === TaskStatus.Failed,
            isRunning: task.status === TaskStatus.Running,
            isNotStarted: task.status === TaskStatus.Init,
            isCancelled: task.status === TaskStatus.Cancelled,
            stepLayer: null,
            lastInLayer: false,
            firstInLayer: false
        };
    }

    private mapStepsToSteps(steps: any[]): StepInfo[] {
        let result: StepInfo[] = [];
        steps.forEach((item) => {
            for (let stepName in item) {
                if (item[stepName]) {
                    const task = JobTreeNode.getChildTaskForStep(this.task, item[stepName]['id']);
                    result.push({
                        name: stepName,
                        value: task,
                        children: task.template.hasOwnProperty('steps') ?
                            this.mapStepsToSteps(task.template.steps) : [],
                        isSkipped: task.status === TaskStatus.Skipped,
                        isSucceeded: task.status === TaskStatus.Success,
                        isFailed: task.status === TaskStatus.Failed,
                        isRunning: task.status === TaskStatus.Running,
                        isNotStarted: task.status === TaskStatus.Init,
                        isCancelled: task.status === TaskStatus.Cancelled,
                        stepLayer: null,
                        lastInLayer: false,
                        firstInLayer: false
                    });
                }
            }
        });
        return result;
    }

    private getFlatListOfSteps(steps: StepInfo[], stepLayer: number): StepInfo[] {
        let result: StepInfo[] = [];
        steps.forEach((item, index) => {
            item.stepLayer = stepLayer == null ? index : stepLayer;
            result.push(item);
            result = result.concat(this.getFlatListOfSteps(item.children, stepLayer == null ? index : stepLayer));
        });
        return result;
    }

    private calculateBoundaries(steps: StepInfo[]): StepInfo[] {
        steps.forEach((item, index) => {
            if (index > 0) {
                if (item.stepLayer !== steps[index - 1].stepLayer) {
                    item.firstInLayer = true;
                    steps[index - 1].lastInLayer = true;
                }
            } else {
                item.firstInLayer = true;
            }
        });
        return steps;
    }

    private showSelectedPanel() {
        if (this.isRecentCommitsVisible) {
            this.recentCommitsComponent.loadRecentCommits(this.task);
        }
        if (this.isJobHistoryVisible) {
            this.jobsHistoryComponent.loadJobsHistory();
        }
        if (this.isJiraCreatePanelVisible) {
            this.jiraService.showJiraIssueCreatorPanel.emit({
                isVisible: true,
                associateWith: 'service',
                itemId: this.task.id,
                name: this.task.name,
                itemUrl: `${location.protocol}//${location.host}/app/timeline/jobs/${this.task.id}`});
        }
        if (this.isJiraIssueListVisible) {
            this.jiraService.showJiraIssuesListPanel.emit({
                isVisible: true,
                associateWith: 'service',
                item: this.task,
                itemUrl: `${location.protocol}//${location.host}/app/timeline/jobs/${this.task.id}`});
        }
    }
}

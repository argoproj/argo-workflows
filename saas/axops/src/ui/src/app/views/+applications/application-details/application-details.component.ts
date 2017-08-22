import { Component, OnInit, OnDestroy } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';
import { Location } from '@angular/common';
import { Subscription, Observable } from 'rxjs';

import { DropdownMenuSettings, NotificationsService } from 'argo-ui-lib/src/components';

import { LayoutSettings, HasLayoutSettings } from '../../layout';
import {
    ApplicationsService, DeploymentsService, JiraService, ModalService, TaskService, ToolService,
} from '../../../services';
import { Application, Commit, Deployment, Pod, ACTIONS_BY_STATUS, APPLICATION_STATUSES, Task } from '../../../model';
import { ViewUtils, LaunchPanelService, GlobalSearchFilters } from '../../../common';

import { STATUS_FILTERS } from '../view-models';

@Component({
    selector: 'ax-app-details',
    templateUrl: './application-details.html',
    styles: [ require('./application-details.scss') ],
})
export class AppDetailsComponent implements OnInit, OnDestroy, LayoutSettings, HasLayoutSettings {

    private appUpdatesSubscription: Subscription = null;
    private subscriptions: Subscription[] = [];

    public tailLogs: boolean = true;
    public application: Application;
    public selectedDeploymentName: string;
    public logsInfo: { pod: Pod, deployment: Deployment } = null;
    public consoleInfo: { pod: Pod, deployment: Deployment } = null;
    public toolbarFilters = {
        data: Object.keys(STATUS_FILTERS).map(key => ({
            name: STATUS_FILTERS[key].title,
            value: key,
            icon: { color: STATUS_FILTERS[key].color },
        })),
        model: [],
        onChange: () => {
            this.router.navigate(['/app/applications/details', this.application.id, ViewUtils.sanitizeRouteParams(this.getRouteParams())]);
        }
    };
    public customStickyPanelHeight = 106;
    public isJiraConfigured: boolean;

    public showSpendingsPanel = false;
    public deploymentAdditionalInfoPanel: { type: 'history' | 'history_details', id: string } = null;

    constructor(
        private router: Router,
        private route: ActivatedRoute,
        private applicationsService: ApplicationsService,
        private deploymentsService: DeploymentsService,
        private modalService: ModalService,
        private notificationsService: NotificationsService,
        private location: Location,
        private launchPanelService: LaunchPanelService,
        private jiraService: JiraService,
        private taskService: TaskService,
        private toolService: ToolService) {
    }

    public ngOnInit() {
        this.route.params.subscribe(params => {
            let applicationId = params['id'];
            this.toolbarFilters.model = (params['filters'] || Object.keys(STATUS_FILTERS).filter(item => item !== 'TERMINATED').join(',')).split(',').filter(item => item !== '');
            if (!this.application || this.application.id !== applicationId) {
                this.loadApplication(applicationId);
            }
            this.selectedDeploymentName = params['deployment'];
            this.logsInfo = this.getDeploymentPodByKey(params['logs']);
            this.consoleInfo = this.getDeploymentPodByKey(params['console']);
            this.showSpendingsPanel = params['spendings'] === 'true';
            let additionalInfo = params['deploymentAdditionalInfo'] || '';
            if (additionalInfo !== '') {
                let [type, id] = additionalInfo.split(':');
                this.deploymentAdditionalInfoPanel = { type, id };
            } else {
                this.deploymentAdditionalInfoPanel = null;
            }
        });
        this.subscriptions.push(this.toolService.isJiraConfigured().subscribe(isConfigured => this.isJiraConfigured = isConfigured));
    }

    public openSelectedDeploymentHistory() {
        this.router.navigate([ ViewUtils.sanitizeRouteParams(this.getRouteParams(), { deploymentAdditionalInfo: 'history'}) ], { relativeTo: this.route });
    }

    public openHistoryDetails(id: string) {
        this.router.navigate([ ViewUtils.sanitizeRouteParams(this.getRouteParams(), { deploymentAdditionalInfo: `history_details:${id}`}) ], { relativeTo: this.route });
    }

    public ngOnDestroy() {
        if (this.subscriptions) {
            this.subscriptions.forEach(subscribtion => subscribtion.unsubscribe());
            this.subscriptions = [];
        }
        this.ensureAppUpdatesUnsubscribed();
    }

    public get pageTitle(): string {
        return this.application && this.application.name || '';
    }

    public get breadcrumb(): { title: string, routerLink?: any[] }[] {
        return [{
            title: 'All Applications',
            routerLink: ['/app/applications']
        }, {
            title: this.application && this.application.name || '',
        }];
    }

    public get selectedDeployment(): Deployment {
        return this.selectedDeploymentName && this.application && this.application.deployments.find(item => item.name === this.selectedDeploymentName);
    }

    public get globalAddActionMenu(): DropdownMenuSettings {
        return this.application && this.applicationMenuCreator(this.application);
    }

    public get podMenuCreator() {
        return (deployment: Deployment, pod: Pod) =>
            new DropdownMenuSettings([{
                title: 'Logs',
                action: () => this.showPodLogs(deployment.name, pod.name),
                iconName: ''
            }, {
                title: 'Console',
                action: () => this.router.navigate(
                    [ ViewUtils.sanitizeRouteParams(this.getRouteParams(), { console: `${deployment.name}:${pod.name}` }) ],
                    { relativeTo: this.route }),
                iconName: ''
            }]);
    }

    public get layoutSettings(): LayoutSettings {
        return this;
    }

    public getFilterClasses(filter) {
        let res = {};
        res[filter.cssClass] = true;
        if (this.toolbarFilters.model.indexOf(filter.key) === -1) {
            res[`${filter.cssClass}-active`] = true;
        }
        return res;
    }

    public clearStatusFilters() {
        this.router.navigate([ViewUtils.sanitizeRouteParams(this.getRouteParams(), { filters: null })], { relativeTo: this.route });
    }

    public toggleStatusFilter(key: string) {
        let filters = this.toolbarFilters.model.slice();
        let index = filters.indexOf(key);
        if (index > -1) {
            filters.splice(index, 1);
        } else {
            filters.push(key);
        }
        this.router.navigate([ViewUtils.sanitizeRouteParams(this.getRouteParams(), { filters: filters.join(',') })], { relativeTo: this.route });
    }

    public onDeploymentSelected(name: string) {
        this.router.navigate([ViewUtils.sanitizeRouteParams( this.getRouteParams() , { deployment: name })], { relativeTo: this.route });
    }

    public showPodLogs(deploymentName: string, podName: string) {
        this.router.navigate(
            [ ViewUtils.sanitizeRouteParams(this.getRouteParams(), { logs: `${deploymentName}:${podName}` }) ],
            { relativeTo: this.route });
    }

    public get applicationMenuCreator() {
        return (application: Application) => {
            let items: {title: string, iconName: string, action: () => any}[] = [];
            if (ACTIONS_BY_STATUS.START.indexOf(application.status) > -1) {
                items.push({
                    title: 'Start', iconName: '',
                    action: () => this.runAction(
                        'Start Application',
                        'Are you sure you want to start application?',
                        'Application has been successfully started',
                        () => this.applicationsService.startApplication(application.id).toPromise()),
                });
            }
            if (ACTIONS_BY_STATUS.STOP.indexOf(application.status) > -1) {
                items.push({
                    title: 'Stop', iconName: '',
                    action: () => this.runAction(
                        'Stop Application',
                        'Are you sure you want to stop application?',
                        'Application has been successfully stopped',
                        () => this.applicationsService.stopApplication(application.id).toPromise()),
                });
            }
            if (ACTIONS_BY_STATUS.TERMINATE.indexOf(application.status) > -1) {
                items.push({
                    title: 'Terminate', iconName: '',
                    action: () => this.runAction(
                        'Terminate Application',
                        'Are you sure you want to terminate application?',
                        'Application has been successfully terminated',
                        () => this.applicationsService.deleteAppById(application.id).toPromise()),
                });
            }
            items.push({
                title: 'Bulk Actions',
                iconName: '',
                action: () => this.navigateToBulkAction(this.application.name),
            });

            if (this.isJiraConfigured) {
               items.push({
                   title: 'Create JIRA Issue',
                   iconName: '',
                   action: () => this.createJiraTicket(this.application),
               });
            }

            items.push({
                title: 'View Spending',
                iconName: '',
                action: () => this.openSpendingsPanel(),
            });
            return new DropdownMenuSettings(items, 'fa-ellipsis-v');
        };
    }

    private async createRedeployTask(deployment: Deployment): Promise<Task> {
        let task = Object.assign(new Task(), { template: deployment.template, parameters: deployment.parameters, template_id: deployment.template_id });
        let stepDependencyRegex = /%%steps[.](.*)[.](.*)%%/g;
        let stepDependendantParams = Object.keys(task.parameters || {}).filter(name => task.parameters[name].match(stepDependencyRegex));
        if (stepDependendantParams.length > 0) {
            let rootTask = await this.taskService.getTask(deployment.task_id).toPromise();
            // Support redeploying with dependencies on artifacts from parent workflow steps.
            stepDependendantParams.forEach(param => {
                let val = task.parameters[param] as string;
                task.parameters[param] = val.replace(stepDependencyRegex, (match, stepName, stepOutput) => {
                    let group = rootTask.template.steps.find(item => !!item[stepName]);
                    if (!group) {
                        throw new Error('Unable to find step which was used to create deployment');
                    }
                    let dependentStep = group[stepName];
                    return `%%artifacts.workflow.${dependentStep.id}.${stepOutput}%%`;
                });
            });
        }
        return task;
    }

    public get deploymentMenuCreator() {
        return (deployment: Deployment) => {
            let items: {title: string, iconName: string, action: () => any}[] = [];
            items.push({
                title: 'Redeploy',  iconName: '',
                action: async () => {
                    let task = await this.createRedeployTask(deployment);
                    let commit = Object.assign(new Commit(), { repo: deployment.template.repo, branch: deployment.template.branch });
                    this.launchPanelService.openPanel(commit, task);
                }
            });
            if (ACTIONS_BY_STATUS.START.indexOf(deployment.status) > -1) {
                items.push({
                    title: 'Start', iconName: '',
                    action: () => this.runAction(
                        'Start Deployment',
                        'Are you sure you want to start deployment?',
                        'Deployment has been successfully started',
                        () => this.deploymentsService.startDeployment(deployment.id).toPromise()),
                });
            }
            if (ACTIONS_BY_STATUS.STOP.indexOf(deployment.status) > -1) {
                items.push({
                    title: 'Stop', iconName: '',
                    action: () => this.runAction(
                        'Stop Deployment',
                        'Are you sure you want to stop deployment?',
                        'Deployment has been successfully stopped',
                        () => this.deploymentsService.stopDeployment(deployment.id).toPromise()),
                });
            }
            if (ACTIONS_BY_STATUS.TERMINATE.indexOf(deployment.status) > -1) {
                items.push({
                    title: 'Terminate', iconName: '',
                    action: () => this.runAction(
                        'Terminate Deployment',
                        'Are you sure you want to terminate deployment?',
                        'Deployment has been successfully terminated',
                        () => this.deploymentsService.deleteDeploymentById(deployment.id).toPromise()),
                });
            }
            return new DropdownMenuSettings(items);
        };
    }

    public onCloseAdditionalInfo() {
        this.router.navigate([ ViewUtils.sanitizeRouteParams(this.getRouteParams(), { deploymentAdditionalInfo: null}) ], { relativeTo: this.route });
    }

    public openSpendingsPanel() {
        this.router.navigate([ ViewUtils.sanitizeRouteParams(this.getRouteParams(), { spendings: 'true'}) ], { relativeTo: this.route });
    }

    private navigateToBulkAction(applicationName) {
        let filters = new GlobalSearchFilters();
        if (applicationName) {
            filters.deployments.app_name = [applicationName];
        }
        this.router.navigate(['/app/search', { category: 'deployments', backRoute: this.location.path(), filters: JSON.stringify(filters)}]);
    }

    private loadApplication(applicationId: string) {
        this.ensureAppUpdatesUnsubscribed();
        this.appUpdatesSubscription = this.applicationsService.getApplicationUpdates(applicationId).flatMap(app => {
            return Observable.fromPromise(Promise.all(
                app.deployments.filter(
                    item => item.previous_deployment_id && item.status === APPLICATION_STATUSES.UPGRADING
                ).map(item => this.deploymentsService.getDeploymentById(item.previous_deployment_id).toPromise()))).map(previousDeployments => {
                    previousDeployments = previousDeployments.filter(item => item.status !== APPLICATION_STATUSES.TERMINATED);
                    let deploymentByName = new Map<string, Deployment>();
                    app.deployments.forEach(item => deploymentByName.set(item.name, item));
                    previousDeployments.forEach(prevDeployment => {
                        let currentDeployment = deploymentByName.get(prevDeployment.name);
                        if (currentDeployment) {
                            currentDeployment.instances = (currentDeployment.instances || []).concat(prevDeployment.instances || []);
                        }
                    });
                    return app;
                });
        }).subscribe(app => this.application = app);
    }

    private ensureAppUpdatesUnsubscribed() {
        if (this.appUpdatesSubscription) {
            this.appUpdatesSubscription.unsubscribe();
            this.appUpdatesSubscription = null;
        }
    }

    private runAction(title: string, confirmation: string, success: string, action: () => Promise<any>) {
        this.modalService.showModal(title, confirmation).subscribe(async confirmed => {
            if (confirmed) {
                await action();
                this.notificationsService.success(success);
            }
        });
    }

    public getLogsSource(deployment: Deployment, pod: Pod) {
        return {
            loadLogs: () => {
                if (pod.containers.length > 0) {
                    return this.deploymentsService.getContainerLiveLog(deployment.id, pod.name, pod.containers[0].name);
                }
                return null;
            },
            getKey: () => `${deployment.id}_${pod.name}_${pod.containers.length}`,
        };
    }

    private getDeploymentPodByKey(key: string) {
        if (key && this.application) {
            let [deploymentName, podName] = key.split(':');
            let deployment = this.application.deployments.find(item => item.name === deploymentName);
            if (deployment) {
                let pod = deployment.instances.find(item => item.name === podName);
                if (pod) {
                    return { pod, deployment };
                }
            }
        }
    }

    private getRouteParams() {
        return { deployment: this.selectedDeploymentName, filters: this.toolbarFilters.model.join(',') };
    }

    public toggleTailLogs() {
        this.tailLogs = !this.tailLogs;
    }

    public closeLogPanel() {
        this.router.navigate([ ViewUtils.sanitizeRouteParams(this.getRouteParams(), { logs: null}) ], { relativeTo: this.route });
    }

    public closeConsolePanel() {
        this.router.navigate([ ViewUtils.sanitizeRouteParams(this.getRouteParams(), { console: null}) ], { relativeTo: this.route });
    }

    public closeSpendingsPanel() {
        this.router.navigate([ ViewUtils.sanitizeRouteParams(this.getRouteParams(), { spendings: null}) ], { relativeTo: this.route });
    }

    public createJiraTicket(application: Application) {
        this.jiraService.showJiraIssueCreatorPanel.emit({
            isVisible: true,
            associateWith: 'application',
            itemId: application.id,
            name: application.name,
            itemUrl: `${location.protocol}//${location.host}/app/applications/details/${application.id}`});
    }
}

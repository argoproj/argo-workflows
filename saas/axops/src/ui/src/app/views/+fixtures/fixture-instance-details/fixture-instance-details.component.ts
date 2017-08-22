import { Component, OnInit, OnDestroy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';

import { Tab, NotificationsService, DropdownMenuSettings, MenuItem } from 'argo-ui-lib/src/components';
import { HasLayoutSettings, LayoutSettings } from '../../layout';
import { FixtureInstance, FixtureClass, TaskFieldNames } from '../../../model';
import { FixtureService, ApplicationsService, TaskService } from '../../../services';

import { FixturesViewService } from '../fixtures.view-service';

@Component({
    selector: 'ax-fixture-instance-details',
    templateUrl: './fixture-instance-details.html',
    styles: [ require('./fixture-instance-details.scss') ],
})
export class FixtureInstanceDetailsComponent implements HasLayoutSettings, LayoutSettings, OnInit, OnDestroy {

    public instance: FixtureInstance = new FixtureInstance();
    public fixtureClass: FixtureClass;
    public activeTab: 'summary' | 'attributes' | 'job_history' = 'summary';
    public isThereAnyAtributes: boolean = false;
    public showEditPanel = false;
    public showClonePanel = false;
    public tasksLoading = true;
    public tasks: any = [];
    public referrers: { name: string, description: string, type: string, routerLink: any[], submitter: string, launch_time: number }[] = [];
    public actionToLaunch: string = '';
    public hasTabs: boolean = true;
    public loadingFixture: boolean = false;
    private subscriptions: Subscription[] = [];

    constructor(private router: Router,
                private route: ActivatedRoute,
                private fixtureService: FixtureService,
                private fixturesViewService: FixturesViewService,
                private notificationsService: NotificationsService,
                private applicationsService: ApplicationsService,
                private taskService: TaskService) {
    }

    public async ngOnInit() {
        this.subscriptions.push(this.fixturesViewService.fixtureUpdated.subscribe(() => {
            this.reloadFixture(this.instance.id);
        }));

        this.route.params.subscribe(async params => {
            let instanceId = params['instanceId'];
            if (this.instance.id !== instanceId) {
                this.reloadFixture(instanceId);
            }
            this.showEditPanel = params['edit'] === 'true';
            this.showClonePanel = params['clone'] === 'true';
            this.activeTab = params['tab'] || 'summary';
            this.actionToLaunch = params['action'] || '';

            if (this.activeTab === 'job_history' && this.tasksLoading) {
                this.getJobsForFixture(instanceId);
            }
        });
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
    }

    public get layoutSettings(): LayoutSettings {
        return this;
    }

    public get pageTitle(): string {
        return this.instance && this.instance.name;
    }

    public get breadcrumb(): { title: string, routerLink?: any[] }[] {
        return [{
            title: 'Fixture Classes',
            routerLink: [ '/app/fixtures' ]
        }, {
            title: this.instance && this.instance.class_name,
            routerLink: [ `/app/fixtures/${this.instance && this.instance.class_id}` ]
        }, {
            title: this.instance && this.instance.name,
        }];
    }

    public get globalAddActionMenu(): DropdownMenuSettings {
        let menuItems: MenuItem[] = [];
        if (this.fixtureClass) {
            menuItems = menuItems.concat(this.fixturesViewService.getInstanceActionMenu(this.instance, this.fixtureClass, {
                customActionLauncher: (actionName, allParamsHasValues) => {
                    if (allParamsHasValues) {
                        this.launchAction(this.instance.id, actionName, null);
                    } else {
                        this.router.navigate([Object.assign(this.getRouterParams(), { action: actionName })], { relativeTo: this.route });
                    }
                },
                cloneLaucher: () => this.router.navigate([Object.assign(this.getRouterParams(), { clone: 'true' })], { relativeTo: this.route }),
                editLaucher: () => this.router.navigate([Object.assign(this.getRouterParams(), {edit: 'true'})], { relativeTo: this.route }),
            }).menu);
        }
        return new DropdownMenuSettings(menuItems, 'fa-ellipsis-v');
    }

    public async toggleFixtureEnabled() {
        await this.fixtureService.setFixtureInstanceEnabled(this.instance.id, !this.instance.enabled);
        this.notificationsService.success(`The fixture '${this.instance.name}' has been updated.`);
        this.reloadFixture(this.instance.id);
    }

    public tabChange(selectedTab: Tab) {
        this.router.navigate(['/app/fixtures', this.instance.class_id, 'details', this.instance.id, {
            tab: selectedTab.tabKey
        }]);
    }

    public hideEditPanel() {
        this.router.navigate(['.', Object.assign(this.getRouterParams(), { edit: 'false' })], { relativeTo: this.route });
    }

    public hideClonePanel() {
        this.router.navigate(['.', Object.assign(this.getRouterParams(), { clone: 'false' })], { relativeTo: this.route });
    }

    public async updateFixture(fixtureInstance: FixtureInstance) {
        await this.fixtureService.updateFixtureInstance(Object.assign(this.instance, fixtureInstance));
        this.notificationsService.success(`The fixture '${this.instance.name}' has been updated.`);
        this.hideEditPanel();
    }

    public async cloneFixture(fixtureInstance: FixtureInstance) {
        fixtureInstance = await this.fixtureService.createFixtureInstance(Object.assign(fixtureInstance, {
            class_id: this.fixtureClass.id,
            class_name: this.fixtureClass.name,
        }));
        this.notificationsService.success(`The fixture '${fixtureInstance.name}' has been created.`);
        this.router.navigate([`/app/fixtures/${this.instance.class_id}/details/${fixtureInstance.id}`]);
    }

    public async closeLaunchPanel(info: { startAction: boolean, fixtureInstanceId?: string, actionName?: string, params?: any }) {
        if (info.startAction) {
            await this.launchAction(info.fixtureInstanceId, info.actionName, info.params);
        }
        this.router.navigate([Object.assign(this.getRouterParams(), { action: '' })], { relativeTo: this.route });
    }

    private async launchAction(fixtureInstanceId: string, actionName: string, params: any) {
        try {
            await this.fixtureService.runFixtureInstanceAction(fixtureInstanceId, actionName, params);
            this.notificationsService.success(`${actionName} job has been successfully started.`);
        } finally {
            this.reloadFixture(this.instance.id);
        }
    }

    private async reloadFixture(instanceId: string) {
        this.loadingFixture = true;
        this.instance = await this.fixtureService.getFixtureInstance(instanceId);
        this.fixtureClass = await this.fixtureService.getFixtureClass(this.instance.class_id);
        this.isThereAnyAtributes = this.instance ? Object.keys(this.instance.attributes).length !== 0 : false;
        this.reloadReferrers();
        this.loadingFixture = false;
    }

    private async reloadReferrers() {
        let referrers = this.instance.referrers || [];

        let appIds = new Set<string>();
        let deploymentIds = new Set<string>();
        referrers.filter(ref => ref.application_generation && ref.deployment_id).forEach(ref => {
            appIds.add(ref.application_generation);
            deploymentIds.add(ref.deployment_id);
        });
        let apps = await Promise.all(Array.from(appIds).map(id => this.applicationsService.getApplicationById(id).toPromise()));
        let deployments = apps.map(app => app.deployments)
            .reduce((first, second) => first.concat(second), [])
            .filter(deployment => deploymentIds.has(deployment.deployment_id));

        let tasks = await Promise.all(referrers.filter(ref => ref.root_workflow_id).map(ref => this.taskService.getTask(ref.root_workflow_id).toPromise()));

        this.referrers = deployments.map(deployment => ({
            name: deployment.name,
            description: deployment.description,
            type: 'deployment',
            routerLink: [`/app/applications/details/${deployment.app_generation}/deployment/${deployment.id}`],
            submitter: deployment.user,
            launch_time: deployment.launch_time,
        })).concat(tasks.map(task => ({
            name: task.name,
            description: task.desc,
            type: 'workflow',
            routerLink: [`/app/timeline/jobs/${task.id}`],
            submitter: task.user,
            launch_time: task.launch_time
        })));
    }

    private async getJobsForFixture(id: string) {
        try {
            this.tasks = await this.taskService.getTasksForFixture(id, [
                TaskFieldNames.name,
                TaskFieldNames.status,
                TaskFieldNames.commit,
                TaskFieldNames.failurePath,
                TaskFieldNames.labels,
                TaskFieldNames.username,
                TaskFieldNames.templateId,
                TaskFieldNames.parameters,
                TaskFieldNames.jira_issues,
                TaskFieldNames.policy_id,
            ]);
        } catch (err) {
            this.tasks = [];
        }

        this.tasksLoading = false;
    }

    private getRouterParams() {
        return {
            tab: this.activeTab,
            edit: String(this.showEditPanel),
            action: this.actionToLaunch || '',
        };
    }
}

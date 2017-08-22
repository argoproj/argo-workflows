import { Component, OnInit, OnDestroy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs';
import { HasLayoutSettings, LayoutSettings } from '../../layout';
import { DropdownMenuSettings, NotificationsService } from 'argo-ui-lib/src/components';
import { FixtureInstance, FixtureClass, FixtureStatuses, FixtureTemplate } from '../../../model';
import { FixtureService, ModalService } from '../../../services';

import { FixturesViewService } from '../fixtures.view-service';

@Component({
    selector: 'ax-fixture-instances',
    templateUrl: './fixture-instances.html',
    styles: [ require('./fixture-instances.scss') ],
})
export class FixtureInstancesComponent implements HasLayoutSettings, LayoutSettings, OnInit, OnDestroy {

    public instances: FixtureInstance[] = [];
    public fixtureClass: FixtureClass = null;
    public showNewInstancePanel: boolean = false;
    public instancePanelMode: string;
    public actionToLaunch: string;
    public reassignTemplates: FixtureTemplate[];
    public loadingFixtureClass: boolean = false;
    public loadingFixtureInstances: boolean = false;
    public toolbarFilters = {
        data: [{
            value: 'active',
            name: 'Active',
            icon: { color: 'success' },
        }, {
            value: 'transient',
            name: 'Transient',
            icon: { color: 'queued' },
        }, {
            value: 'deleted',
            name: 'Deleted',
            icon: { color: 'fail' },
        }],
        model: [],
        onChange: (data) => {
            this.router.navigate([{ filters: data.join(',') }], { relativeTo: this.route });
        }
    };

    private selectedFixtureId: string;
    private classId: string = null;
    private subscriptions: Subscription[] = [];

    constructor(
        private router: Router,
        private route: ActivatedRoute,
        private fixtureService: FixtureService,
        private fixturesViewService: FixturesViewService,
        private modalService: ModalService,
        private notificationsService: NotificationsService) {
    }

    public ngOnInit() {
        this.route.params.subscribe(async params => {
            let classId = params['id'];
            if (this.classId !== classId) {
                this.classId = classId;
                this.loadingFixtureClass = true;
                this.fixtureClass = await this.fixtureService.getFixtureClass(this.classId);
                this.loadingFixtureClass = false;
                this.reloadFixtures();
            }
            this.showNewInstancePanel = params['create'] === 'true';
            this.selectedFixtureId = params['instanceId'];
            this.instancePanelMode = null;
            if (this.selectedFixtureId) {
                if (params['clone'] === 'true') {
                    this.instancePanelMode = 'clone';
                } else if (params['edit'] === 'true') {
                    this.instancePanelMode = 'edit';
                }
            }
            if (params['reassign'] === 'true') {
                if (!this.reassignTemplates) {
                    this.reassignTemplates = await this.fixtureService.getFixtureTemplates();
                }
            } else {
                this.reassignTemplates = null;
            }
            this.toolbarFilters.model = (params['filters'] === undefined ? 'active,transient' : params['filters']).split(',');
            this.actionToLaunch = params['action'] || '';
        });
        this.subscriptions.push(this.fixturesViewService.fixtureUpdated.asObservable().subscribe(async fixture => {
            this.reloadFixtures();
        }));
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(item => item.unsubscribe());
        this.subscriptions = [];
    }

    public async closeLaunchPanel(info: { startAction: boolean, fixtureInstanceId?: string, actionName?: string, params?: any }) {
        if (info.startAction) {
            await this.launchAction(info.fixtureInstanceId, info.actionName, info.params);
        }
        this.router.navigate([Object.assign(this.getRouteParams(), { action: '', instanceId: '' })], { relativeTo: this.route });
    }

    public get selectedFixtureInstance(): FixtureInstance {
        return this.selectedFixtureId && this.instances.find(item => item.id === this.selectedFixtureId);
    }

    public get filteredInstances() {
        let transientStatuses = [
            FixtureStatuses.DELETE_ERROR,
            FixtureStatuses.CREATE_ERROR,
            FixtureStatuses.CREATING,
            FixtureStatuses.DELETING,
            FixtureStatuses.INIT,
            FixtureStatuses.OPERATING
        ];
        return this.instances.filter(item => {
            return (item.status === FixtureStatuses.ACTIVE && this.toolbarFilters.model.indexOf('active') > -1)
                || (item.status === FixtureStatuses.DELETED && this.toolbarFilters.model.indexOf('deleted') > -1)
                || (transientStatuses.indexOf(item.status) > -1 && this.toolbarFilters.model.indexOf('transient') > -1);
        });
    }

    public async createOrEditInstance(instance: FixtureInstance) {
        if (this.instancePanelMode === 'edit' && this.selectedFixtureId) {
            await this.fixtureService.updateFixtureInstance(Object.assign(this.selectedFixtureInstance, instance));
        } else {
            await this.fixtureService.createFixtureInstance(Object.assign(instance, {
                class_id: this.classId,
                class_name: this.fixtureClass.name,
            }));
        }
        this.hideNewInstancePanel();
        this.reloadFixtures();
    }

    public get layoutSettings(): LayoutSettings {
        return this;
    }

    public get globalAddActionMenu(): DropdownMenuSettings {
        return new DropdownMenuSettings([{
            title: 'Create Instance',
            action: () => this.router.navigate([{ create: 'true' }], { relativeTo: this.route }),
            iconName: ''
        }, {
            title: 'Reassign Template',
            action: () => this.router.navigate([{ reassign: 'true' }], { relativeTo: this.route }),
            iconName: ''
        }, {
            title: 'Delete Class',
            action: () => {
                this.modalService.showModal('Delete fixture class?', 'Are you sure you want to delete fixture class?').subscribe(async (confirmed) => {
                    if (confirmed) {
                        await this.fixtureService.deleteFixtureClass(this.classId);
                        this.router.navigate(['..'], { relativeTo: this.route });
                    }
                });
            },
            iconName: ''
        }], 'fa-ellipsis-v');
    }

    public async closeReassignPanel(info: { selectedTemplateId: string }) {
        this.router.navigate(['.', { reassign: 'false' }], { relativeTo: this.route });
        if (info.selectedTemplateId) {
            await this.fixtureService.updateFixtureClass(this.classId, info.selectedTemplateId);
            this.reloadFixtures();
        }
    }

    public getFixtureAttributes(fixture: FixtureInstance) {
        return Object.keys(this.fixtureClass.attributes || {}).map(key => ({
            name: key,
            value: fixture.attributes[key],
            required: (this.fixtureClass.attributes[key].flags || '').split(',').indexOf('required')
        })).sort((first, second) => second.required - first.required).slice(0, 5);
    }

    public hideInstancePanel() {
        this.router.navigate([{ clone: 'false', edit: 'false' }], { relativeTo: this.route });
    }

    public hideNewInstancePanel() {
        this.router.navigate([{ create: 'false' }], { relativeTo: this.route });
    }

    public get pageTitle(): string {
        return this.fixtureClass && this.fixtureClass.name;
    }

    public get breadcrumb(): { title: string, routerLink?: any[] }[] {
        return [{
            title: 'Fixture Classes',
            routerLink: [ '/app/fixtures' ]
        }, {
            title: this.fixtureClass ? this.fixtureClass.name : '',
        }];
    }

    public getActionMenu(instance: FixtureInstance): DropdownMenuSettings {
        return this.fixturesViewService.getInstanceActionMenu(instance, this.fixtureClass, {
            customActionLauncher: (actionName, allParamsHasValues) => {
                if (allParamsHasValues) {
                    this.launchAction(instance.id, actionName, null);
                } else {
                    this.router.navigate([Object.assign(this.getRouteParams(), { action: actionName, instanceId: instance.id })], { relativeTo: this.route });
                }
            },
            cloneLaucher: () => this.router.navigate([Object.assign(this.getRouteParams(), { clone: 'true', instanceId: instance.id })], { relativeTo: this.route }),
            editLaucher: () => this.router.navigate([Object.assign(this.getRouteParams(), { edit: 'true', instanceId: instance.id })], { relativeTo: this.route }),
        });
    }

    public trackByFixtureInstanceId(instance: FixtureInstance) {
        return instance.id;
    }

    public trackByAttributeName(attribute) {
        return attribute.name;
    }

    private async reloadFixtures() {
        this.loadingFixtureInstances = true;
        this.instances = await this.fixtureService.getFixtureInstances(this.classId);
        this.loadingFixtureInstances = false;
    }

    private getRouteParams() {
        return { filters: this.toolbarFilters.model.join(',') };
    }

    private async launchAction(fixtureInstanceId: string, actionName: string, params: any) {
        try {
            await this.fixtureService.runFixtureInstanceAction(fixtureInstanceId, actionName, params);
            this.notificationsService.success(`${actionName} job has been successfully started.`);
        } finally {
            this.reloadFixtures();
        }
    }

}

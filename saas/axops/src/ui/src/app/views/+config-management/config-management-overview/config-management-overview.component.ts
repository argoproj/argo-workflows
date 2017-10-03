import { Component, OnInit } from '@angular/core';
import { HasLayoutSettings, LayoutSettings } from '../../layout';
import { Configuration, ViewPreferences } from '../../../model';
import { AuthenticationService, ConfigsService, ModalService, ViewPreferencesService } from '../../../services';
import { ViewUtils } from '../../../common';
import { Router, ActivatedRoute } from '@angular/router';
import { DropdownMenuSettings, NotificationsService } from 'argo-ui-lib/src/components';

@Component({
    selector: 'ax-config-management-overview',
    templateUrl: './config-management-overview.html',
    styles: [ require('./config-management-overview.scss') ]
})

export class ConfigManagementOverviewComponent implements OnInit, HasLayoutSettings, LayoutSettings {

    public configurations: Configuration[] = [];
    public loading: boolean;
    public selectedConfig: Configuration;
    public currentUser: string;
    public create: boolean;

    private viewPreferences: ViewPreferences;
    private showMyOnly: boolean;

    constructor(
        private router: Router,
        private activatedRoute: ActivatedRoute,
        private authenticationService: AuthenticationService,
        private configsService: ConfigsService,
        private modalService: ModalService,
        private notificationsService: NotificationsService,
        private viewPreferencesService: ViewPreferencesService,
    ) {}

    public async ngOnInit() {
        this.currentUser = this.authenticationService.getUsername();
        this.viewPreferences = await this.viewPreferencesService.getViewPreferences();
        this.activatedRoute.params.subscribe(async params => {
            let viewPreferencesFilterState = this.viewPreferences.filterStateInPages['/app/config-management'] || {};
            let edit = params['edit'] || '';
            this.create = false;
            if (edit) {
                let [user, name]: [string, string] = edit.split(':');
                this.selectedConfig = await this.configsService.getUserConfiguration(user, name, true);
            } else if (params['createNew'] === 'true' || params['createNew'] === 'secret' ) {
                this.selectedConfig = { is_secret: params['createNew'] === 'secret' };
                this.create = true;
            } else {
                this.selectedConfig = null;
            }

            let showMyOnly = params['showMyOnly'] ?
                params['showMyOnly'] === 'true' : viewPreferencesFilterState.filters.indexOf('myown') > -1;
            if (this.showMyOnly !== showMyOnly) {
                this.toolbarFilters.model = showMyOnly ? ['myown'] : [];
                this.showMyOnly = showMyOnly;
                this.loadConfigurations();

                this.viewPreferencesService.updateViewPreferences(v => {
                    v.filterStateInPages['/app/config-management'] = {
                        filters: this.toolbarFilters.model,
                    };
                });
            }
        });
    }

    public get layoutSettings(): LayoutSettings {
        return this;
    }

    public pageTitle = 'Configurations';

    public get globalAddActionMenu(): DropdownMenuSettings {
        return new DropdownMenuSettings([{
            title: 'Add New Config as Kubernetes Secret',
            action: () => {
                this.router.navigate(
                    ['/app/config-management', ViewUtils.sanitizeRouteParams(this.getRouteParams(), { createNew: 'secret' })], { relativeTo: this.activatedRoute });
            },
            iconName: '',
        }, {
            title: 'Add New Public Config',
            action: () => {
                this.router.navigate(
                    ['/app/config-management', ViewUtils.sanitizeRouteParams(this.getRouteParams(), { createNew: 'true' })], { relativeTo: this.activatedRoute });
            },
            iconName: '',
        }], 'fa fa-plus');
    }

    public getPanelMode(config: Configuration) {
        if (this.create) {
            return 'create';
        } else if (config && this.currentUser === config.user) {
            return 'edit';
        } else {
            return 'view';
        }
    }

    public toolbarFilters = {
        data: [{
            name: 'My Configurations',
            value: 'myown',
        }],
        model: [],
        onChange: () => {
            this.router.navigate(['/app/config-management',
                ViewUtils.sanitizeRouteParams(this.getRouteParams(), { showMyOnly: !this.showMyOnly })], { relativeTo: this.activatedRoute });
        }
    };

    public async loadConfigurations() {
        this.loading = true;
        this.configurations = await this.configsService.getConfigurations({ user: this.showMyOnly ? this.currentUser : null });
        this.configurations.sort( (a, b) => {
            if ((a.user === this.currentUser) && (b.user !== this.currentUser)) {
               return -1;
            } else if ((a.user !== this.currentUser) && (b.user === this.currentUser)) {
                return 1;
            } else {
                if (a.ctime >  b.ctime) {
                    return -1;
                } else {
                    return 1;
                }
            }
        });
        this.loading = false;
    }

    public getDropdownMenu(config) {
        if (config.user === this.currentUser) {
            return new DropdownMenuSettings([{
                title: 'Edit',
                iconName: 'fa fa-pencil-square-o',
                action: () => this.editSelectedConfig(config)
            }, {
                title: 'Delete',
                iconName: 'fa-times-circle-o',
                action: () => this.deleteConfig(config)
            }]);
        } else {
            return new DropdownMenuSettings([{
                title: 'Read',
                iconName: 'fa fa-book',
                action: () => this.editSelectedConfig(config)
            }]);
        }
    }

    public editSelectedConfig(config: Configuration) {
        this.router.navigate([
            '/app/config-management', ViewUtils.sanitizeRouteParams(this.getRouteParams(), { edit: `${config.user}:${config.name}` })], { relativeTo: this.activatedRoute });
    }

    public async deleteConfig(config: Configuration) {
        this.modalService.showModal('Delete configuration', `Are you sure you want to delete configuration '${config.name}'`).subscribe(async confirmed => {
            if (confirmed) {
                await this.configsService.deleteConfiguration( {user: this.currentUser, name: config.name} );
                this.notificationsService.success('Configuration has been deleted successfully.');
                this.loadConfigurations();
            }
        });
    }

    public closePanel() {
        this.router.navigate([
            '/app/config-management', ViewUtils.sanitizeRouteParams(this.getRouteParams(), { createNew: false, edit: null })], { relativeTo: this.activatedRoute });
    }

    public async saveConfigChange(config: Configuration) {
        config = Object.assign(config, { user: this.currentUser });
        if (this.create) {
            await this.configsService.createConfiguration(config);
            this.notificationsService.success('Configuration has been created successfully.');
        } else {
            await this.configsService.updateConfiguration(config);
            this.notificationsService.success('Configuration has been updated successfully.');
        }
        this.loadConfigurations();
        this.closePanel();
    }

    private getRouteParams() {
        return { showMyOnly: this.showMyOnly.toString() };
    }
}

import { Component, OnInit } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';

import { LayoutSettings, HasLayoutSettings } from '../../layout';
import { Volume, Deployment } from '../../../model';
import { VolumesService, ModalService, ApplicationsService } from '../../../services';
import { ShortDateTimePipe } from '../../../pipes';
import { DateRange, DropdownMenuSettings, NotificationsService } from 'argo-ui-lib/src/components';

@Component({
    selector: 'ax-volume-details',
    templateUrl: './volume-details.html',
    styles: [ require('./volume-details.scss') ]
})
export class VolumeDetailsComponent implements OnInit, HasLayoutSettings, LayoutSettings {
    public showEditPanel = false;
    public showChartLoader: boolean = true;
    public volumeLoader: boolean = false;
    public volume: Volume = new Volume();
    public editedVolume: Volume;
    public dateRange = DateRange.today();
    public deployments: Deployment[] = [];
    public charts: { title: string, data, options }[] = [];

    constructor(
        private route: ActivatedRoute,
        private router: Router,
        private volumesService: VolumesService,
        private applicationsService: ApplicationsService,
        private notificationsService: NotificationsService,
        private modalService: ModalService) {}

    public ngOnInit() {
        this.route.params.subscribe(async params => {
            this.showEditPanel = params['edit'] === 'true';
            this.layoutDateRange.data = params['date'] ? DateRange.fromRouteParams(params, -1) : DateRange.today();

            let volumeChanged = this.volume.id !== params['id'];
            if (volumeChanged) {
                this.volumeLoader = true;
                this.volume = await this.volumesService.getVolumeById(params['id']);
                this.volumeLoader = false;
                this.editedVolume = this.volume.clone();

                let appIds = new Set<string>();
                let deploymentIds = new Set<string>();
                this.volume.referrers.forEach(ref => {
                    appIds.add(ref.application_generation);
                    deploymentIds.add(ref.deployment_id);
                });
                let apps = await Promise.all(Array.from(appIds).map(id => this.applicationsService.getApplicationById(id).toPromise()));
                this.deployments = apps.map(app => app.deployments)
                    .reduce((first, second) => first.concat(second), [])
                    .filter(deployment => deploymentIds.has(deployment.deployment_id));
            }
            let dateRange = DateRange.fromRouteParams(params);
            if (!DateRange.equals(this.dateRange, dateRange) || volumeChanged) {
                this.dateRange = dateRange;
                this.loadStats();
            }
        });
    }

    public layoutDateRange = {
        data: DateRange.today(),
        onApplySelection: (date) => {
            this.router.navigate(['.', date.toRouteParams()], { relativeTo: this.route } );
        },
        isAllDates: false
    };

    public get layoutSettings(): LayoutSettings {
        return this;
    }

    public cancelEdit() {
        this.router.navigate(['.', { edit: 'false' }], { relativeTo: this.route } );
    }

    public async save() {
        this.volume = await this.volumesService.updateVolume(this.editedVolume);
        this.editedVolume = this.volume.clone();
        this.notificationsService.success('Volume has been successfully updated.');
        this.router.navigate(['.', { edit: 'false' }], { relativeTo: this.route } );
    }

    private setInterval() {
        // if timerange less than 7 days set hour interval, if timerange longer set daily
        return (this.dateRange.endDate.unix() - this.dateRange.startDate.unix()) < (86400 * 7) ? 3600 : 86400;
    }

    private async loadStats() {
        let interval = this.setInterval();

        this.showChartLoader = true;
        let readopsStats = await this.volumesService.getChartStats(this.volume.id, 'readops', interval, this.dateRange.startDate, this.dateRange.endDate);
        let writeopsStats = await this.volumesService.getChartStats(this.volume.id, 'writeops', interval, this.dateRange.startDate, this.dateRange.endDate);
        let readtotStats = await this.volumesService.getChartStats(this.volume.id, 'readtot', interval, this.dateRange.startDate, this.dateRange.endDate);
        let writetotStats = await this.volumesService.getChartStats(this.volume.id, 'writetot', interval, this.dateRange.startDate, this.dateRange.endDate);
        let readSizeAvgStats = await this.volumesService.getChartStats(this.volume.id, 'readsizeavg', interval, this.dateRange.startDate, this.dateRange.endDate);
        this.showChartLoader = false;

        let gbFormatter = v => `${v.toFixed(2)} GiB`;
        this.charts = [];
        let chartsMeta = [
            {title: 'IOPS', xFormatter: gbFormatter, props: [{title: 'READ', name: 'read'}, {title: 'WRITE', name: 'write'}], data: [readopsStats, writeopsStats]},
            {title: 'LATENCY',
                xFormatter: v => `${v.toFixed()} sec.`,
                props: [{title: 'READ', name: 'latency_read'}, {title: 'WRITE', name: 'latency_write'}], data: [readtotStats, writetotStats]},
            {title: 'SIZE', xFormatter: gbFormatter, props: [{title: 'SIZE', name: 'size'}], data: [readSizeAvgStats]}];
        let colors = ['#E2C2DC', '#C1EDF0'];
        for (let meta of chartsMeta) {
            this.charts.push({
                title: meta.title,
                options: {
                    chart: {
                        type: 'lineChart',
                        height: 116,
                        clipEdge: false,
                        margin: { top: 0, left: -1, right: 0, bottom: 6 },
                        yAxis: { tickFormat: meta.xFormatter },
                        xAxis: { tickFormat: v => new ShortDateTimePipe().transform(v, []) }
                    }
                },
                data: meta.props.map((prop, i) => ({
                    values: meta.data[i] ? meta.data[i].map(item => ({x: item[0], y: item[1]})) : [],
                    key: prop.title,
                    strokeWidth: 3,
                    color: colors[i % colors.length],
                }))
            });
        }
    }

    get globalAddActionMenu(): DropdownMenuSettings {
        if (this.volume && this.volume.anonymous) {
            return null;
        }

        return new DropdownMenuSettings([{
            title: 'Edit',
            iconName: 'fa-pencil-square-o',
            action: async () => {
                this.router.navigate(['.', { edit: 'true' }], { relativeTo: this.route });
            }
        }, {
            title: 'Delete',
            iconName: 'fa-times-circle-o',
            action: async () => {
                this.modalService.showModal('Delete Volume', `Are you sure you want to delete volume '${this.volume.name}'?`).subscribe(async confirmed => {
                    if (confirmed) {
                        await this.volumesService.deleteVolume(this.volume.id);
                        this.notificationsService.success('Volume has been successfully deleted.');
                        this.router.navigate(['..'], { relativeTo: this.route });
                    }
                });
            }
        }], 'fa-ellipsis-v');
    }

    get breadcrumb(): { title: string, routerLink: any[] }[] {
        return [{
            title: 'Volumes',
            routerLink: [ '/app/volumes' ]
        }];
    }

    get pageTitle(): string {
        return this.volume && this.volume.name || '';
    }
}

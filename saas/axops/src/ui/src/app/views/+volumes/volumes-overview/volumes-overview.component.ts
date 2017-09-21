import { Component, OnInit } from '@angular/core';
import { Router, ActivatedRoute } from '@angular/router';

import { HasLayoutSettings, LayoutSettings } from '../../layout';
import { VolumesService } from '../../../services';
import { Volume } from '../../../model';
import { NotificationsService } from 'argo-ui-lib/src/components';

@Component({
    selector: 'ax-volumes-overview',
    templateUrl: './volumes-overview.component.html',
    styles: [ require('./volumes-overview.scss') ]
})
export class VolumesOverviewComponent implements HasLayoutSettings, LayoutSettings, OnInit {

    public providers: { type: string, volumes: Volume[] }[] = [];
    public showAddPanel: boolean = false;
    public volumesType: 'named' | 'anonymous';
    public editedVolume: Volume;
    public showEditPanel = false;
    public editVolumeId: string;

    constructor(
        private volumesService: VolumesService,
        private router: Router,
        private route: ActivatedRoute,
        private notificationsService: NotificationsService) {
    }

    public get layoutSettings(): LayoutSettings {
        return {
            pageTitle: 'Volumes',
            globalAddAction: () => {
                this.router.navigate(['/app/volumes', { add: 'true' }], { relativeTo: this.route });
            },
            hasTabs: true,
            breadcrumb: [{
                title: 'All Volumes',
                routerLink: null,
            }],
        };
    }

    public ngOnInit() {
        this.route.params.subscribe(async params => {
            this.showAddPanel = params['add'] === 'true';
            this.showEditPanel = params['edit'] === 'true';
            this.editVolumeId = params['volumeId'];
            let volumesType = params['type'] || 'named';
            if (this.volumesType !== volumesType) {
                this.volumesType = volumesType;
                this.loadVolumes();
            }
        });
    }

    public changeVolumesType(volumeType: string) {
        this.router.navigate(['/app/volumes', { type: volumeType }], { relativeTo: this.route });
    }

    public closeAddPanel(info: { isVolumeCreated: boolean }) {
        this.router.navigate(['/app/volumes', { add: 'false' }], { relativeTo: this.route });
        if (info.isVolumeCreated) {
            this.loadVolumes();
        }
    }

    public onDeletedVolume() {
        this.loadVolumes();
    }

    public onEditVolume(volume: Volume) {
        this.editedVolume = volume;
        this.router.navigate(['/app/volumes', { edit: 'true', volumeId: volume.id }], { relativeTo: this.route });
    }

    public cancelEdit() {
        this.router.navigate(['/app/volumes', { edit: 'false' }], { relativeTo: this.route } );
    }

    public async editVolume(volume: Volume) {
        await this.volumesService.updateVolume(volume);
        this.notificationsService.success('Volume has been successfully updated.');
        this.router.navigate(['/app/volumes', { edit: 'false' }], { relativeTo: this.route } );

        this.loadVolumes();
    }

    private async loadVolumes() {
        let volumes = await this.volumesService.getVolumes({ named: this.volumesType === 'named' }, false);
        let volumesByProvider = new Map<string, Volume[]>();
        volumes.forEach(volume => {
            let providerVolumes = volumesByProvider.get(volume.storage_provider) || [];
            providerVolumes.push(volume);
            volumesByProvider.set(volume.storage_provider, providerVolumes);
        });
        this.providers = Array.from(volumesByProvider.entries()).map(entry => ({
            type: entry[0],
            volumes: entry[1]
        }));
    }
}

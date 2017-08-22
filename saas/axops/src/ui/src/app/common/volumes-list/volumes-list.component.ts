import { Component, Input, Output, EventEmitter } from '@angular/core';

import { DropdownMenuSettings, NotificationsService } from 'argo-ui-lib/src/components';

import { VolumesService, ModalService } from '../../services';
import { Volume } from '../../model';

@Component({
    selector: 'ax-volumes-list',
    templateUrl: './volumes-list.html',
    styles: [ require('./volumes-list.scss') ],
})
export class VolumesListComponent {

    @Input()
    public volumes: Volume[];

    @Output()
    public onDeletedVolume: EventEmitter<string> = new EventEmitter<string>();

    constructor(
        private volumesService: VolumesService,
        private notificationsService: NotificationsService,
        private modalService: ModalService) {
    }

    public getDropdownMenu(volume: Volume) {
        return new DropdownMenuSettings([{
            title: 'Delete',
            iconName: 'fa-times-circle-o',
            action: () => {
                this.modalService.showModal('Delete Volume', `Are you sure you want to delete volume '${volume.name}'?`).subscribe(async confirmed => {
                    if (confirmed) {
                        await this.volumesService.deleteVolume(volume.id);
                        this.notificationsService.success('Volume deletion has been started.');
                        this.onDeletedVolume.next(volume.id);
                    }
                });
            }
        }]);
    }
}

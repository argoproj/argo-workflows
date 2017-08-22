import { Component, OnInit, Input, EventEmitter, Output, ViewChild } from '@angular/core';

import { NotificationsService } from 'argo-ui-lib/src/components';
import { Volume, StorageClass } from '../../../model';
import { VolumesService } from '../../../services';
import { VolumeFormWidgetComponent } from '../volume-form-widget/volume-form-widget.component';

@Component({
    selector: 'ax-volume-add-panel',
    templateUrl: './volume-add-panel.html',
    styles: [ require('./volume-add-panel.scss') ]
})
export class VolumeAddPanelComponent implements OnInit {

    public volume: Volume = new Volume();
    public selectedStorageClass: StorageClass;
    public storageClasses: StorageClass[];

    @Input()
    public get show(): boolean {
        return this.showPanel;
    }

    public set show(showPanel: boolean) {
        if (this.showPanel !== showPanel) {
            this.showPanel = showPanel;
            this.volume = new Volume();
            this.selectedStorageClass = null;
        }
    }

    @Output()
    public onClosePanel = new EventEmitter<{ isVolumeCreated: boolean }>();

    @ViewChild(VolumeFormWidgetComponent)
    public volumeFormWidget: VolumeFormWidgetComponent;

    private showPanel = false;

    constructor(private volumesService: VolumesService, private notificationsService: NotificationsService) {}

    public ngOnInit() {
        this.volumesService.getStorageClasses().then(res => this.storageClasses = res);
    }

    public selectStorageClass(storageClass: StorageClass) {
        this.selectedStorageClass = storageClass;
    }

    public closePanel(isVolumeCreated = false) {
        this.onClosePanel.emit({ isVolumeCreated });
    }

    public get formInvalid(): boolean {
        return this.volumeFormWidget && this.volumeFormWidget.volumeForm ? this.volumeFormWidget.volumeForm.invalid : false;
    }

    public async save() {
        await this.volumesService.createVolume(this.volumeFormWidget.volume.name, this.volumeFormWidget.volume.attributes.size_gb, this.selectedStorageClass);
        this.closePanel(true);
        this.notificationsService.success('Volume has been successfully created.');
    }
}

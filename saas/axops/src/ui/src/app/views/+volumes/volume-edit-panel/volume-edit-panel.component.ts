import { Component, Input, Output, EventEmitter, ViewChild, OnInit } from '@angular/core';

import { Volume } from '../../../model';
import { VolumeFormWidgetComponent } from '../volume-form-widget/volume-form-widget.component';
import { VolumesService } from '../../../services/volumes.service';

@Component({
    selector: 'ax-volume-edit-panel',
    templateUrl: './volume-edit-panel.html',
})
export class VolumeEditPanelComponent implements OnInit {
    public loaderEditVolume: boolean = false;

    @Output()
    public closePanel = new EventEmitter<any>();
    @Output()
    public saveVolume = new EventEmitter<Volume>();

    @Input()
    public show: boolean;
    @Input()
    public volume: Volume;
    @Input()
    public editVolumeId: string;

    @ViewChild(VolumeFormWidgetComponent)
    public volumeFormWidget: VolumeFormWidgetComponent;

    constructor(private volumesService: VolumesService) {
    }

    ngOnInit() {
        if (!this.volume && !!this.editVolumeId) {
            this.getVolumeById(this.editVolumeId);
        }
    }

    public close() {
        this.closePanel.emit({});
    }

    public save() {
        this.saveVolume.emit(this.volume);
    }

    public get formInvalid(): boolean {
        return this.volumeFormWidget && this.volumeFormWidget.volumeForm ? this.volumeFormWidget.volumeForm.invalid : false;
    }

    private async getVolumeById(volumeId) {
        this.loaderEditVolume = true;
        this.volume = await this.volumesService.getVolumeById(volumeId);
        this.loaderEditVolume = false;
    }
}

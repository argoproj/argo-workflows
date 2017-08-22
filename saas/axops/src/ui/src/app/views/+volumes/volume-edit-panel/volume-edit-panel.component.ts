import { Component, Input, Output, EventEmitter, ViewChild } from '@angular/core';

import { Volume } from '../../../model';
import { VolumeFormWidgetComponent } from '../volume-form-widget/volume-form-widget.component';

@Component({
    selector: 'ax-volume-edit-panel',
    templateUrl: './volume-edit-panel.html',
})
export class VolumeEditPanelComponent {
    @Output()
    public closePanel = new EventEmitter<any>();
    @Output()
    public saveVolume = new EventEmitter<Volume>();

    @Input()
    public show: boolean;
    @Input()
    public volume: Volume;

    @ViewChild(VolumeFormWidgetComponent)
    public volumeFormWidget: VolumeFormWidgetComponent;

    public close() {
        this.closePanel.emit({});
    }

    public save() {
        this.saveVolume.emit(this.volume);
    }

    public get formInvalid(): boolean {
        return this.volumeFormWidget && this.volumeFormWidget.volumeForm ? this.volumeFormWidget.volumeForm.invalid : false;
    }
}

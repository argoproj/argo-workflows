import { Component, Input, ViewChild } from '@angular/core';

import { Volume, StorageClass } from '../../../model';
import { NgForm } from '@angular/forms';

@Component({
    selector: 'ax-volume-form-widget',
    templateUrl: './volume-form-widget.html',
    styles: [ require('./volume-form-widget.scss') ]
})
export class VolumeFormWidgetComponent {

    @Input()
    public storageClass: StorageClass;
    @Input()
    public volume: Volume;

    @ViewChild('volumeForm')
    public volumeForm: NgForm;
}

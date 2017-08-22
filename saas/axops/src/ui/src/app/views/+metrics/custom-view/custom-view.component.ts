import * as _ from 'lodash';
import { Component, Input, Output, EventEmitter } from '@angular/core';
import { FormGroup, FormControl, Validators } from '@angular/forms';

import { CustomViewService } from '../../../services';
import { CustomView } from '../../../model';

@Component({
    selector: 'ax-custom-view',
    templateUrl: './custom-view.html',
    styles: [ require('./custom-view.scss') ],
})
export class CustomViewComponent {

    @Output()
    closePopup = new EventEmitter<any>();

    public submitted: boolean;

    private customViewForm: FormGroup;
    private customView: CustomView;
    private isEdit: boolean = false;

    @Input()
    showClose: boolean = true;

    @Input()
    set setCustomView(value: CustomView) {
        this.customView = value;
        this.isEdit = (value && value.id !== '');
        this.submitted = false;
        if (value && this.customViewForm) {
            (<FormControl>this.customViewForm.controls['name']).setValue(value.name);
        }
    }

    constructor(private customViewService: CustomViewService) {
        if (!this.customView) {
            this.customView = new CustomView();
        }
        this.customViewForm = new FormGroup({
            name: new FormControl(this.customView.name, Validators.required)
        });
    }

    /**
     * Perform the create/update action for custom view being created
     */
    saveCustomView(customViewForm: FormGroup) {
        this.submitted = true;
        if (customViewForm.valid) {
            // if the name has changed, new view should be created
            if (this.isEdit && (customViewForm.value.name !== this.customView.name)) {
                this.isEdit = false;
                this.customView.id = null;
            }

            this.customView = _.assign(this.customView, customViewForm.value);

            if (!this.isEdit) {
                this.customViewService.createCustomView(this.customView).subscribe(data => {
                    this.close(data);
                });
            } else {
                this.customViewService.updateCustomView(this.customView).subscribe(data => {
                    this.close(data);
                });
            }
        }
    }

    close(data?: CustomView) {
        this.submitted = false;
        this.customViewForm.reset({
            name: this.customView.name ? this.customView.name : '',
        });
        this.closePopup.emit(data ? data : false);
    }
}

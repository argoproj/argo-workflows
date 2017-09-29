import { Component, EventEmitter, Input, Output } from '@angular/core';
import { FormGroup, FormArray, FormControl, Validators, AbstractControl } from '@angular/forms';

import { Configuration } from '../../../model';
import { CustomRegex } from '../../../common';

@Component({
    selector: 'ax-config-management-panel',
    templateUrl: './config-management-panel.html',
    styles: [ require('./config-management-panel.scss') ],
})

export class ConfigManagementPanelComponent {

    private config: Configuration;
    private panelMode: 'create' | 'edit' | 'view' = 'create';

    public title: string;
    public instructText: string;
    public namePattern = CustomRegex.templateInputName;
    public shownItems: number[] = [];

    @Input()
    public set selectedConfig(config: Configuration) {
        this.config = config;
        this.refreshForm();
        this.refreshTitles();
    }

    @Input()
    public openPanel: boolean;

    @Input()
    public set mode(val: 'create' | 'edit' | 'view') {
        this.panelMode = val;
        this.refreshForm();
        this.refreshTitles();
    }

    @Output()
    public onSave = new EventEmitter<Configuration>();

    @Output()
    public onCancel = new EventEmitter<any>();

    public configForm: FormGroup;

    constructor() {
        this.configForm = new FormGroup({
            name: new FormControl(null, Validators.required),
            description: new FormControl(null),
            values: new FormArray([], (values: FormArray) => {
                let nonUniqueIndexes = [];
                let keys = new Set<string>();
                values.controls.forEach((item: FormGroup, i) => {
                    let key = (item.controls.key.value || '').trim();
                    if (key) {
                        if (keys.has(key)) {
                            nonUniqueIndexes.push(i);
                        } else {
                            keys.add(key);
                        }
                    }
                });
                if (nonUniqueIndexes.length === 0) {
                    return null;
                }
                let errors = {};
                nonUniqueIndexes.forEach(i => errors['nonUniqueKeys-' + i] = true);
                return errors;
            })
        });
    }

    public addValue(key: string, value: string, enabled = true) {
        let values = this.configForm.controls.values as FormArray;
        let keyControl = new FormControl(key, [Validators.required, Validators.pattern(CustomRegex.templateInputName)]);
        this.setEnabled(keyControl, enabled);
        let valueControl = new FormControl(value, [Validators.required]);
        this.setEnabled(valueControl, enabled);
        values.push(new FormGroup({
            key: keyControl,
            value: valueControl,
        }));
    }

    public showValue(i: number) {
        let index = this.shownItems.indexOf(i);
        if (index > -1) {
            this.shownItems.splice(index, 1);
        } else {
            this.shownItems.push(i);
        }
    }

    public isShown(i: number) {
        return this.shownItems.indexOf(i) > -1;
    }

    public save() {
        let formValue = this.configForm.value;
        let configValue = {};
        (formValue.values || []).forEach(item => {
            configValue[item.key] = item.value;
        });
        this.onSave.emit({
            name: formValue.name || this.config.name,
            description: formValue.description || ' ',
            is_secret: this.config.is_secret,
            value: configValue
        });
    }

    public cancel() {
        this.onCancel.emit();
    }

    public removeKey(i: number) {
        let valuesControls = this.configForm.controls['values'] as FormArray;
        valuesControls.removeAt(i);
    }

    private refreshForm() {
        this.configForm.reset();
        this.configForm.controls['name'].setValue(this.config && this.config.name);
        this.setEnabled(this.configForm.controls['name'], this.panelMode === 'create');
        this.configForm.controls['description'].setValue(this.config && this.config.description);
        this.setEnabled(this.configForm.controls['description'], this.panelMode === 'create' || this.panelMode === 'edit');
        let valuesControls = this.configForm.controls['values'] as FormArray;
        while (valuesControls.length > 0) {
            valuesControls.removeAt(0);
        }
        Object.keys(this.config && this.config.value || {}).forEach(key => {
            this.addValue(key, this.config.value[key], this.panelMode !== 'view');
        });
        this.shownItems = [];
    }

    private setEnabled(control: AbstractControl, enabled: boolean) {
        if (enabled) {
            control.enable();
        } else {
            control.disable();
        }
    }

    private refreshTitles() {
        this.instructText = 'Please enter data that you want to save as Configuration';
        switch (this.panelMode) {
            case 'create':
                this.title = this.config && this.config.is_secret ? 'Add New Config as Kubernetes Secret' : 'Add New Public Config';
                break;
            case 'edit':
                this.title = this.config && this.config.name;
                break;
            case 'view':
                this.title = this.config && this.config.name;
                break;
        }
    }
}

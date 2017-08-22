import { Component, Input, Output, EventEmitter, ViewChild } from '@angular/core';
import { FixtureClass, FixtureInstance } from '../../../model';
import { FixtureInstanceFormComponent } from '../fixture-instance-form/fixture-instance-form.component';

@Component({
    selector: 'ax-fixture-instance-attributes-panel',
    templateUrl: './fixture-instance-attributes-panel.html'
})
export class FixtureInstanceAttributesPanelComponent {

    @ViewChild('fixtureForm')
    public fixtureForm: FixtureInstanceFormComponent;
    @Input()
    public mode: 'create' | 'edit' | 'clone' = 'create';
    @Input()
    public fixtureClass: FixtureClass;
    @Input()
    public set fixtureInstance (val: FixtureInstance) {
        this.instance = val;
        this.instanceForClone = Object.assign(new FixtureInstance(), val, { name: null });
    }

    @Input()
    public set show(val: boolean) {
        if (val !== this.visible) {
            this.visible = val;
            if (!val && this.fixtureForm) {
                this.fixtureForm.reset();
            }
        }
    }

    @Output()
    public closePanel = new EventEmitter();
    @Output()
    public save = new EventEmitter<FixtureInstance>();

    public visible = false;
    public instanceForClone: FixtureInstance;
    public instance: FixtureInstance;

    public get messages() {
        switch (this.mode) {
            case 'create':
                return {
                    title: 'Create New Fixture',
                    saveButtonTitle: 'Create',
                };
            case 'edit':
                return {
                    title: 'Edit Fixture',
                    saveButtonTitle: 'Save',
                };
            case 'clone':
                return {
                    title: 'Clone Fixture',
                    saveButtonTitle: 'Clone',
                };
        }
    }

    public hidePanel() {
        this.closePanel.emit({});
    }

    public processInstance(instance: FixtureInstance) {
        this.save.emit(instance);
    }
}

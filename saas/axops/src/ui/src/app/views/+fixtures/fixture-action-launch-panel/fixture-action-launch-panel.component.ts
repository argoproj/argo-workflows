import { Component, Input, Output, EventEmitter } from '@angular/core';
import { FormGroup, FormControl, Validators } from '@angular/forms';

import { NotificationsService } from 'argo-ui-lib/src/components';

import { FixtureClass, FixtureInstance } from '../../../model';
import { FixtureService } from '../../../services';


@Component({
    selector: 'ax-fixture-action-launch-panel',
    templateUrl: './fixture-action-launch-panel.html',
    styles: [ require('./fixture-action-launch-panel.scss') ]
})
export class FixtureActionLaunchPanelComponent {

    public form: FormGroup;
    public parameters: { name: string, value: string }[] = [];
    public show: boolean;
    public actionName: string;

    @Output()
    public close = new EventEmitter<{ startAction: boolean, fixtureInstanceId?: string, actionName?: string, params?: any }>();

    private fixtureInstanceId: string;

    constructor(private fixtureService: FixtureService, private notificationsService: NotificationsService) {}

    @Input()
    public set actionInfo(val: {actionName: string, fixtureClass: FixtureClass, fixtureInstance: FixtureInstance }) {
        if (!val || !val.fixtureClass || !val.fixtureInstance || !val.actionName) {
            this.show = false;
            return;
        }
        this.show = true;
        this.fixtureInstanceId = val.fixtureInstance.id;
        this.actionName = val.actionName;

        let action = val.fixtureClass.actions[val.actionName];
        if (!action) {
            return;
        }
        let templateName = action.template;
        let template = val.fixtureClass.action_templates[templateName];
        if (!template || !template.inputs || !template.inputs.parameters) {
            return;
        }
        this.parameters = Object.keys(template.inputs.parameters || {}).map(name => ({
            name,
            value: action.parameters[name] || template.inputs.parameters[name].default || ''
        }));
        this.form = new FormGroup({});
        this.parameters.forEach(param => {
            this.form.addControl(param.name, new FormControl(param.value, [Validators.required]));
        });
    }

    public async submitAction() {
        this.close.emit({ startAction: true, fixtureInstanceId: this.fixtureInstanceId, actionName: this.actionName, params: this.form.value });
    }

    public hidePanel() {
        this.close.emit({ startAction: false });
    }
}

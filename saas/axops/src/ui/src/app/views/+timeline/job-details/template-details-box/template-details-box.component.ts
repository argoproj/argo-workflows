import * as _ from 'lodash';
import { Component, Input } from '@angular/core';

import { Task } from '../../../../model';

@Component({
    selector: 'ax-template-details-box',
    templateUrl: './template-details-box.html',
})
export class TemplateDetailsBoxComponent {

    @Input()
    public set setTask(value: Task) {
        if (value) {
            this.task = value;
            this.parametersList = this.getParameterList(value);
        }
    }

    private task: Task;
    private parametersList: Object[] = [];

    getParameterList(task: Task) {
        let parametersList = [];
        _.forOwn(task.arguments, (value, key) => {
            parametersList.push({
                name: key,
                value: value
            });
        });
        return parametersList;
    }
}

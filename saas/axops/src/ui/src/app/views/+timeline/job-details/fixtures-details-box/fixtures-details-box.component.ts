import * as _ from 'lodash';
import { Component, Input } from '@angular/core';

import { Task } from '../../../../model';

@Component({
    selector: 'ax-fixtures-details-box',
    templateUrl: './fixtures-details-box.html',
})
export class FixturesDetailsBoxComponent {

    @Input()
    public set setTask(value: Task) {
        if (value) {
            this.task = value;
            this.fixtures = this.getFixturesList(value);
        }
    }

    private task: Task;
    private fixtures: any[] = [];

    getFixturesList(task) {
        let fixturesList = [];
        if (task.template.fixtures) {
            task.template.fixtures.forEach(f => {

                _.forOwn(f, (value, key) => {
                    fixturesList.push({
                        name: key,
                        value: value
                    });
                });
            });
        }
        if (task.template.steps) {
            task.template.steps.forEach(s => {
                _.forOwn(s, (value, key) => {
                    fixturesList = fixturesList.concat(this.getFixturesList(value));
                });
            });
        }

        return fixturesList;
    }
}

import { Component, Input } from '@angular/core';
import * as _ from 'lodash';

import { Task } from '../../../../model';
import { TaskService } from '../../../../services';
import { JobTreeNode, UsedFixtureInfoWithSteps } from '../../../../common/workflow-tree/workflow-tree.view-models';

type FixtureInfo = UsedFixtureInfoWithSteps & { chartData: { y: number }[] };

@Component({
    selector: 'ax-job-fixtures-box',
    templateUrl: './job-fixtures-box.html',
    styles: [ require('./job-fixtures-box.scss') ]
})
export class JobFixturesBoxComponent {

    public chartOptions = {
        chart: {
            type: 'pieChart',
            height: 150,
            margin: { top: 0, left: 0, right: 0, bottom: 0 },
            showLabels: false,
            duration: 500,
            labelThreshold: 0.01,
            labelSunbeamLayout: true,
            showLegend: false,
            donut: true,
            donutRatio: 0.70,
            tooltip: { enabled: false },
            color: [
                // used color
                '#1ABC9C',
                // not used
                '#CCD6DD'
            ]
        }
    };

    @Input()
    public set task (val: Task) {
        if (!_.isEqual(val, new Task)) {
            let tree = JobTreeNode.createFromTask(val);
            let totalNumberOfSteps = tree.getFlattenNodes().length;
            this.fixtures = tree.getAllUsedFixtures().map(
                fixture => Object.assign({}, fixture, { chartData: [{y: fixture.steps.length}, {y: totalNumberOfSteps - fixture.steps.length}] }));
            this.fixturesLoader = false;
        } else {
            this.fixturesLoader = true;
        }
    }

    public fixtures: FixtureInfo[] = [];
    public selectedFixture: FixtureInfo = null;
    public fixturesLoader: boolean = false;

    constructor(private taskService: TaskService) {}

    public selectFixture(fixture: FixtureInfo) {
        this.selectedFixture = fixture;
    }

    public getFixtureAttributes(fixture: FixtureInfo) {
        return Object.keys(fixture.staticFixtureInfo || {}).map(key => ({
            name: key,
            value: fixture.staticFixtureInfo[key],
        })).slice(0, 5);
    }

    public trackByAttributeName(attribute) {
        return attribute.name;
    }

    public trackByFixtureId(fixture: FixtureInfo) {
        return fixture.name;
    }

    getLogsSource(task: Task) {
        return {
            loadLogs: () => {
                return this.taskService.getTaskLogs(task.id);
            },
            getKey() {
                return `${task.id}_${task.status}`;
            }
        };
    }
}

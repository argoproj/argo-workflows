import { Component, Input } from '@angular/core';

import { TaskStatus, Task } from '../../../model';
import { CentsToDollarsPipe, ShortDateTimePipe, DurationPipe } from '../../../pipes';
import { NodeInfo } from '../../../common/workflow-tree/workflow-tree.view-models';

@Component({
    selector: 'ax-job-step-summary',
    templateUrl: './job-step-summary.html',
    styles: [ require('./job-step-summary.scss') ]
})
export class JobStepSummaryComponent {

    public errorDetailsCollapsed = false;
    public task: Task;

    public get isFailedTask() {
        return this.task && this.task.status === TaskStatus.Failed;
    }

    @Input()
    public set step(val: NodeInfo) {
        let task = val.workflow.value;
        this.task = task;

        this.attributes = [{
            name: 'Name',
            value: val.name
        }, {
            name: 'Description',
            value: task.desc,
        }, {
            name: 'Cost',
            value: new CentsToDollarsPipe().transform(task.cost, 2)
        }];
        if (task.template.type === 'container' && task.template.resources) {
            this.attributes = this.attributes.concat([{
                name: 'Memory used',
                value: `${task.template.resources.mem_mib} mb`,
            }, {
                name: 'CPU used',
                value: task.template.resources.cpu_cores.toString(),
            }]);
        }
        this.attributes = this.attributes.concat([{
            name: 'Created at',
            value: new ShortDateTimePipe().transform(task.create_time, []),
        }, {
            name: 'Launched at',
            value: new ShortDateTimePipe().transform(task.launch_time, []),
        }, {
            name: 'Init time',
            value: new DurationPipe().transform(task.init_time, true),
        }, {
            name: 'Wait time',
            value: new DurationPipe().transform(task.wait_time, true),
        }, {
            name: 'Run time',
            value: new DurationPipe().transform(task.run_time, true),
        }]);
        if (task.status === TaskStatus.Failed || task.status === TaskStatus.Success) {
            this.attributes.push({
                name: task.status === TaskStatus.Failed ? 'Failed at' : 'Completed at',
                value: new ShortDateTimePipe().transform(task.launch_time + task.run_time, []),
            });
        }
    }

    public attributes: { name: string, value: string }[] = [];
}
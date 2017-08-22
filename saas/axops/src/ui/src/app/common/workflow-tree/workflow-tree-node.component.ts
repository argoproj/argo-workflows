import { Component, Input, Output, EventEmitter, TemplateRef } from '@angular/core';
import { NodeInfo } from './workflow-tree.view-models';
import { TaskStatus, Task } from '../../model';

@Component({
    selector: 'ax-workflow-tree-node',
    templateUrl: './workflow-tree-node.html',
})
export class WorkflowTreeNodeComponent {
    @Input()
    public rootTask: Task;
    @Input()
    public activeStep: string = '';
    @Input()
    public cell: NodeInfo;
    @Input()
    public hasDownConnectorArrow: boolean;
    @Input()
    public workflowNodeActionsTemplate: TemplateRef<any>;
    @Input()
    public workflowNodeInfoTemplate: TemplateRef<any>;
    @Input()
    public clickable = false;

    @Output()
    public onSelectTask: EventEmitter<any> = new EventEmitter();

    @Output()
    public onGetYaml: EventEmitter<NodeInfo> = new EventEmitter<NodeInfo>();

    public statuses = TaskStatus;

    selectTask(node: NodeInfo) {
        this.onSelectTask.next(node);
    }

    getYaml(task) {
        this.onGetYaml.next(task);
    }
}

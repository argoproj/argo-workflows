import { Component, Input, Output, EventEmitter, ContentChild, TemplateRef } from '@angular/core';

import { Task, Template } from '../../model';
import { JobTreeNode, NodeInfo } from './workflow-tree.view-models';

@Component({
    selector: 'ax-workflow-tree',
    templateUrl: './workflow-tree.html',
    styles: [ require('./workflow-tree.scss') ],
})
export class WorkflowTreeComponent {
    @Input()
    public activeStep: string = '';

    @Output()
    public onSelectNode: EventEmitter<NodeInfo> = new EventEmitter<NodeInfo>();

    @Output()
    public onClickShowYaml: EventEmitter<NodeInfo> = new EventEmitter<NodeInfo>();

    @ContentChild('nodeActions')
    public workflowNodeActionsTemplate: TemplateRef<any>;

    @ContentChild('nodeInfo')
    public workflowNodeInfoTemplate: TemplateRef<any>;

    @Input()
    set task(value: Task) {
        this.nodeGroups = value ? JobTreeNode.createFromTask(value).children : [];
    }

    @Input()
    set template(template: Template) {
        this.nodeGroups = template ? JobTreeNode.createFromTemplate(template).children : [];
    }

    @Input()
    public nodeClickable = false;

    public nodeGroups: JobTreeNode[][] = [];
    public expandedTaskIds: string[] = [];

    public selectTask(node: NodeInfo) {
        this.onSelectNode.emit(node);
    }

    public clickShowYaml(node: NodeInfo) {
        this.onClickShowYaml.emit(node);
    }

    public trackByIndex(index: number) {
        return index;
    }
}

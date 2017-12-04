import { Component, Input, Output, EventEmitter, ContentChild, TemplateRef } from '@angular/core';

import * as models from '../../models';
import { NodeInfo, WorkflowTree } from './workflow-tree.view-models';

@Component({
  selector: 'ax-workflow-tree',
  templateUrl: './workflow-tree.html',
  styleUrls: ['./workflow-tree.scss'],
})
export class WorkflowTreeComponent {
  @Input()
  public activeStep = '';

  @Output()
  public onSelectNode: EventEmitter<NodeInfo> = new EventEmitter<NodeInfo>();

  @Output()
  public onClickShowYaml: EventEmitter<NodeInfo> = new EventEmitter<NodeInfo>();

  @ContentChild('nodeActions')
  public workflowNodeActionsTemplate: TemplateRef<any>;

  @ContentChild('nodeInfo')
  public workflowNodeInfoTemplate: TemplateRef<any>;

  @Input()
  set workflowTree(value: WorkflowTree) {
    this.nodeGroups = value ? value.root.children : [];
  }

  @Input()
  public nodeClickable = false;

  public nodeGroups: NodeInfo[][] = [];
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

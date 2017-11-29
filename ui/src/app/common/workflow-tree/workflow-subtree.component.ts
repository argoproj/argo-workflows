import { Component, Input, Output, EventEmitter, TemplateRef, ElementRef, AfterViewChecked, NgZone } from '@angular/core';
import { NodeInfo } from './workflow-tree.view-models';
import * as models from '../../models';
import * as $ from 'jquery';

@Component({
  selector: 'ax-workflow-subtree',
  templateUrl: './workflow-subtree.html',
})
export class WorkflowSubtreeComponent implements AfterViewChecked {

  public nodesInfo: NodeInfo[] = [];

  @Input()
  public activeStep = '';
  @Input()
  public hasRightConnector: boolean;
  @Input()
  public hasDownConnectorArrow: boolean;
  @Input()
  public hideTreeConnector = false;
  @Input()
  public rootTask: models.Workflow;
  @Input()
  public expandedTaskIds: string[] = [];
  @Input()
  public nodeClickable = false;

  @Input()
  public set nodes(value: NodeInfo[]) {
    this.nodesInfo = value.map(node => Object.assign(node, { expanded: false }));
  }

  @Input()
  public workflowNodeActionsTemplate: TemplateRef<any>;
  @Input()
  public workflowNodeInfoTemplate: TemplateRef<any>;

  @Output()
  public onSelectTask: EventEmitter<any> = new EventEmitter();

  @Output()
  public onGetYaml: EventEmitter<NodeInfo> = new EventEmitter<NodeInfo>();

  constructor(private el: ElementRef, private zone: NgZone) { }

  public ngAfterViewChecked() {
    this.zone.runOutsideAngular(() => {
      this.refreshNodeConnectorHeight();
    });
  }

  public toggleExpanded(info: NodeInfo) {
    const index = this.expandedTaskIds.indexOf(this.getNodeUniqueId(info));
    if (index > -1) {
      this.expandedTaskIds.splice(index, 1);
    } else {
      this.expandedTaskIds.push(this.getNodeUniqueId(info));
    }
  }

  public isExpanded(info: NodeInfo) {
    return this.expandedTaskIds.indexOf(this.getNodeUniqueId(info)) > -1;
  }

  public selectTask(node: NodeInfo) {
    this.onSelectTask.next(node);
  }

  public getNodeUniqueId(info: NodeInfo) {
    return info.stepName;
  }

  public refreshNodeConnectorHeight() {
    const expandedNodes = $('> div > div.clearfix > .workflow-tree__node-expanded', this.el.nativeElement).toArray();
    expandedNodes.forEach(node => {
      const expandedNode = $(node);
      expandedNode.find('> div.workflow-tree__node-container').toArray().forEach(subTreeEl => {
        const connector = $(subTreeEl).find('.workflow-tree__node-down-connector').first();
        const firstSubNode = $('> ax-workflow-subtree > div > .clearfix:last-of-type > .fl:first-of-type', subTreeEl);
        connector.css({ height: firstSubNode.offset().top - connector.offset().top + 120 });
      });
    });
  }

  public getYaml(node: NodeInfo): void {
    this.onGetYaml.next(node);
  }
}

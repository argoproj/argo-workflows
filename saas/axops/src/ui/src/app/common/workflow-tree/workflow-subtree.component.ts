import { Component, Input, Output, EventEmitter, TemplateRef, ElementRef, AfterViewChecked, NgZone } from '@angular/core';
import { JobTreeNode, NodeInfo } from './workflow-tree.view-models';
import { Task } from '../../model';

@Component({
    selector: 'ax-workflow-subtree',
    templateUrl: './workflow-subtree.html',
})
export class WorkflowSubtreeComponent implements AfterViewChecked {

    public nodesInfo: NodeInfo[] = [];

    @Input()
    public activeStep: string = '';
    @Input()
    public hasRightConnector: boolean;
    @Input()
    public hasDownConnectorArrow: boolean;
    @Input()
    public hideTreeConnector: boolean = false;
    @Input()
    public rootTask: Task;
    @Input()
    public expandedTaskIds: string[] = [];
    @Input()
    public nodeClickable = false;

    @Input()
    public set nodes(value: JobTreeNode[]) {
        this.nodesInfo = value.map((node, i) => {
            return  {
                name: node.name,
                workflow: node,
                expanded: false,
            };
        });
    }

    @Input()
    public workflowNodeActionsTemplate: TemplateRef<any>;
    @Input()
    public workflowNodeInfoTemplate: TemplateRef<any>;

    @Output()
    public onSelectTask: EventEmitter<any> = new EventEmitter();

    @Output()
    public onGetYaml: EventEmitter<NodeInfo> = new EventEmitter<NodeInfo>();

    constructor(private el: ElementRef, private zone: NgZone) {}

    public ngAfterViewChecked() {
        this.zone.runOutsideAngular(() => {
            this.refreshNodeConnectorHeight();
        });
    }

    public toggleExpanded(info: NodeInfo) {
        let index = this.expandedTaskIds.indexOf(this.getNodeUniqueId(info));
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
        return info.name + '_' + info.workflow.name;
    }

    public refreshNodeConnectorHeight() {
        let expandedNodes = $('> div > div.clearfix > .workflow-tree__node-expanded', this.el.nativeElement).toArray();
        expandedNodes.forEach(node => {
            let expandedNode = $(node);
            expandedNode.find('> div.dib').toArray().forEach(subTreeEl => {
                let connector = $(subTreeEl).find('.workflow-tree__node-down-connector').first();
                let firstSubNode = $('> ax-workflow-subtree > div > .clearfix:last-of-type > .fl:first-of-type', subTreeEl);
                connector.css({ height: firstSubNode.offset().top - connector.offset().top + 120 });
            });
        });
    }

    public getYaml(node: NodeInfo): void {
        this.onGetYaml.next(node);
    }
}

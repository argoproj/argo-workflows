import * as moment from 'moment';
import { Component, OnInit, OnDestroy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs/Subscription';

import { WorkflowsService } from '../../services';
import * as models from '../../models';
import { NodeInfo, WorkflowTree } from '../../common';
import { Observable } from 'rxjs/Observable';
import { race } from 'rxjs/operators';

@Component({
  selector: 'ax-workflow-details-page',
  templateUrl: './workflow-details-page.html',
  styleUrls: ['./workflow-details.scss']
})
export class WorkflowDetailsPageComponent implements OnInit, OnDestroy {

  private subscriptions: Subscription[] = [];

  public tree: WorkflowTree;
  public workflow: models.Workflow;
  public selectedTab = 'summary';
  public stepDetailsPanel: { nodeName: string; tab: string; } = null;

  constructor(private workflowsService: WorkflowsService, private route: ActivatedRoute, private router: Router) {}

  public tabChange(tab: string) {
    this.router.navigate(['.', { tab }], { relativeTo: this.route });
  }

  public ngOnInit() {
    const treeSrc = this.route.params.map(params => params['name']).distinctUntilChanged()
        .flatMap(name => this.workflowsService.getWorkflowStream(name)).map(workflow => new WorkflowTree(workflow)).share();

    this.subscriptions.push(treeSrc.subscribe(tree => {
      this.workflow = tree.workflow;
      this.tree = tree;
    }));

    this.subscriptions.push(this.route.params.map(params => params['tab']).distinctUntilChanged().subscribe(tab => {
      this.selectedTab = tab || 'summary';
    }));

    this.subscriptions.push(Observable.combineLatest(
      treeSrc, this.route.params.map(params => params['node'] || '').distinctUntilChanged()).subscribe(([tree, node]) => {
        if (node) {
          const [nodeName, tab] = node.split(':');
          this.stepDetailsPanel = { nodeName, tab };
        } else {
          this.stepDetailsPanel = null;
        }
      }));

    this.subscriptions.push(Observable.combineLatest(treeSrc, Observable.interval(1000)).subscribe(() => {
      if (this.workflow) {
        Object.keys(this.workflow.status.nodes)
            .map(name => this.workflow.status.nodes[name]).filter(node => node.startedAt).forEach(node => {
          const endTime = node.finishedAt ? moment(node.finishedAt) : moment();
          node['runDuration'] = endTime.diff(moment(node.startedAt)) / 1000;
        });
      }
    }));
  }

  public ngOnDestroy() {
    this.subscriptions.forEach(s => s.unsubscribe());
    this.subscriptions = [];
  }

  public getProgressClasses(node: NodeInfo) {
    const stepStatus = node.status.phase;
    const status = stepStatus === models.NODE_PHASE.FAILED ? 'failed' : 'running';
    let percentage = 0;
    switch (stepStatus) {
      case models.NODE_PHASE.RUNNING:
        const avgDuration = 60;
        percentage = Math.min(((node.status['runDuration'] || 0) / avgDuration) * 100, 95);
        break;
      case models.NODE_PHASE.SUCCEEDED:
      case models.NODE_PHASE.SKIPPED:
      case models.NODE_PHASE.FAILED:
      case models.NODE_PHASE.ERROR:
        percentage = 100;
        break;
    }
    return [
        'workflow-details__node-progress', `workflow-details__node-progress--${percentage.toFixed()}-${status}`
    ].join(' ');
  }

  public showStepDetails(stepName: string, detailsTab: string = 'logs') {
    this.router.navigate(['.', { tab: this.selectedTab, node: `${stepName}:${detailsTab}` }], { relativeTo: this.route });
  }

  public closeStepDetailsPanel() {
    this.router.navigate(['.', { tab: this.selectedTab }], { relativeTo: this.route });
  }

  public time(dateTime: string) {
    return moment(dateTime).format('H:mm:ss');
  }
}

import * as moment from 'moment';
import { Component, OnInit, OnDestroy } from '@angular/core';
import { ActivatedRoute, Router } from '@angular/router';
import { Subscription } from 'rxjs/Subscription';

import { WorkflowsService, EventsService, SystemService } from '../../services';
import * as models from '../../models';
import { NodeInfo, WorkflowTree } from '../../common';
import { Observable } from 'rxjs/Observable';
import { race } from 'rxjs/operators';
import { DropdownMenuSettings } from 'ui-lib/src/components';
import { NODE_PHASE } from '../../models';

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
  public actionSettingsByNodeName = new Map<String, DropdownMenuSettings>();
  public selectedYamlStep: string;
  public isYamlVisible: boolean;
  public isConsoleVisible: boolean;
  public consoleNodeName: string;
  public isWebConsoleEnabled: boolean;

  constructor(private workflowsService: WorkflowsService,
              private route: ActivatedRoute,
              private router: Router,
              private eventsService: EventsService,
              private systemService: SystemService) {}

  public tabChange(tab: string) {
    this.router.navigate(['.', { tab }], { relativeTo: this.route });
  }

  public async ngOnInit() {
    this.eventsService.setPageTitle.emit(this.route.snapshot.params.name);
    const settings = await this.systemService.getSettings();
    this.isWebConsoleEnabled = settings.isWebConsoleEnabled;
    const treeSrc = this.route.params
        .distinctUntilChanged((first, second) => first['name'] === second['name'] && first['namespace'] === second['namespace'] )
        .flatMap(params => this.workflowsService.getWorkflowStream(
          params['namespace'], params['name'])).map(workflow => new WorkflowTree(workflow)).share();

    this.subscriptions.push(treeSrc.subscribe(tree => {
      this.workflow = tree.workflow;
      this.tree = tree;
      this.regenerateActionMenuSettings();
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

    this.subscriptions.push(this.route.params.map(params => params['yaml'] || '').distinctUntilChanged().subscribe(yaml => {
      if (yaml) {
        this.selectedYamlStep = yaml;
        this.isYamlVisible = true;
      } else {
        this.selectedYamlStep = null;
        this.isYamlVisible = false;
      }
    }));

    this.subscriptions.push(this.route.params.map(params => params['console'] || '').distinctUntilChanged().subscribe(ssh => {
      this.isConsoleVisible = !!ssh;
      if (this.isConsoleVisible) {
        this.consoleNodeName = ssh;
      }
    }));
    this.subscriptions.push(Observable.combineLatest(treeSrc, Observable.interval(1000)).subscribe(() => {
      if (this.tree) {
        Object.keys(this.tree.workflowStatusNodes)
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

  public showYaml(node: NodeInfo) {
    this.router.navigate(['.', { tab: this.selectedTab, yaml: node.stepName }], { relativeTo: this.route });
  }

  public showConsole(nodeName: string) {
    this.router.navigate(['.', { tab: this.selectedTab, console: nodeName }], { relativeTo: this.route });
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

  private regenerateActionMenuSettings() {
    this.actionSettingsByNodeName = new Map<String, DropdownMenuSettings>();
    for (const artifact of this.tree.getArtifacts()) {
      let settings = this.actionSettingsByNodeName.get(artifact.nodeName);
      if (!settings) {
        settings = new DropdownMenuSettings([]);
        this.actionSettingsByNodeName.set(artifact.nodeName, settings);
      }
      settings.menu.push({
        title: `Download artifact '${artifact.name}'`,
        action: () => window.open(artifact.downloadUrl),
        iconName: '',
      });
    }

    for (const nodeName of Object.keys(this.tree.workflowStatusNodes)) {
      let settings = this.actionSettingsByNodeName.get(nodeName);
      if (!settings) {
        settings = new DropdownMenuSettings([]);
        this.actionSettingsByNodeName.set(nodeName, settings);
      }
      const status = (this.tree.workflowStatusNodes)[nodeName];
      if (status.phase === NODE_PHASE.RUNNING && this.isWebConsoleEnabled) {
        settings.menu.push({
          title: 'View Console',
          action: () => this.showConsole(nodeName),
          iconName: 'fa-terminal',
        });
      }
    }
  }
}

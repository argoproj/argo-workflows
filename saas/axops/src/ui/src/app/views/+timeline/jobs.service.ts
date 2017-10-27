import * as _ from 'lodash';
import { Router } from '@angular/router';
import { EventEmitter, Output, Injectable } from '@angular/core';
import { LocationStrategy } from '@angular/common';
import { DomSanitizer } from '@angular/platform-browser';

import { Task, TaskStatus } from '../../model';
import { DropdownMenuSettings, MenuItem, NotificationsService } from 'argo-ui-lib/src/components';
import { TaskService, ModalService, DeploymentsService, SystemService, AuthenticationService, ArtifactsService } from '../../services';
import { JobTreeNode } from '../../common/workflow-tree/workflow-tree.view-models';
import { LaunchPanelService } from '../../common/multiple-service-launch-panel/launch-panel.service';

export interface TaskArtifact {
    name: string;
    browsable: boolean;
}

@Injectable()
export class JobsService {
    @Output()
    updateJobsList: EventEmitter<any> = new EventEmitter();

    @Output()
    setCurrentJobPosition: EventEmitter<any> = new EventEmitter();

    @Output()
    showJob: EventEmitter<Task> = new EventEmitter<Task>();

    private isPlayground: boolean;
    private isAdmin: boolean;

    constructor(
        private router: Router,
        private notificationsService: NotificationsService,
        private taskService: TaskService,
        private launchPanelService: LaunchPanelService,
        private modalService: ModalService,
        private deploymentsService: DeploymentsService,
        private domSanitizer: DomSanitizer,
        private locationStrategy: LocationStrategy,
        private systemService: SystemService,
        private authenticationService: AuthenticationService,
        private artifactsService: ArtifactsService,
    ) {
        this.systemService.isPlayground().then(isPlayground => this.isPlayground = isPlayground);
        this.authenticationService.getCurrentUser().then(user => this.isAdmin = user.isAdmin() || user.isSuperAdmin());
    }

    getTaskArtifacts(task: Task): TaskArtifact[] {
        let artifacts: TaskArtifact[] = [];

        if (!task ||
            !task.hasOwnProperty('template') ||
            !task.template.hasOwnProperty('outputs') ||
            !task.template.outputs.hasOwnProperty('artifacts')) {
            return null;
        }

        _.forOwn(task.template.outputs.artifacts, (v, k) => {
            artifacts.push({
                name: k,
                browsable: v.meta_data && v.meta_data.indexOf('browsable') > -1
            });
        });

        return artifacts;
    }

    public getSelectedStep(task: Task, rootTask: Task) {
        let steps = JobTreeNode.createFromTask(rootTask).getFlattenNodes();
        let step = steps.find(node => node.value.id === task.id);
        if (!step) {
            throw `Task '${task.id}' is not child of task '${rootTask.id}'`;
        }
        return step;
    }

    public getActionMenuSettings(task: Task, rootTask: Task): DropdownMenuSettings {
        let step = this.getSelectedStep(task, rootTask);

        let menuItems: MenuItem[] = [];

        if ((!this.isPlayground || this.isAdmin) && task.status === TaskStatus.Running && task.template.type === 'container') {
            menuItems.push({
                title: 'View Ax Console',
                iconName: 'fa-terminal',
                action: () => this.router.navigate([`/app/timeline/jobs/${rootTask.id}`, {
                    tab: 'workflow',
                    consoleStep: step.value.id
                }])
            });
        }

        let dnsName = this.findHostMapping(task);
        if (dnsName) {
            menuItems.push({
                title: 'DNS Name',
                iconName: 'fa-location-arrow',
                action: () => {
                    this.modalService.copyModal('DNS Name', `${dnsName}`, null, null, true);
                }
            });
        }

        if (task.template && task.template.type === 'deployment' && task.status === TaskStatus.Success) {
            menuItems.push({
                title: 'View Deployment',
                iconName: 'ax-icon-deployment',
                action: () => {
                    this.deploymentsService.getDeploymentById(task.id).subscribe(deployment => {
                        this.router.navigateByUrl(`/app/applications/details/${deployment.app_generation}/deployment/${deployment.id}`);
                    });
                }
            });
        }

        let actionMenu: DropdownMenuSettings = new DropdownMenuSettings(menuItems);
        actionMenu.icon = 'fa-ellipsis-v';
        return actionMenu;
    }

    findHostMapping(task: Task) {
        // TODO (alexander): Remove this temporal solution for AppDynamics after deployment feature release
        if (task.template && task.template['labels'] && task.template['labels'].ax_ea_deployment) {
            try {
                let hostMapping: string = JSON.parse(task.template['labels'].ax_ea_deployment).host_mapping;
                if (hostMapping) {
                    Object.keys(task.arguments).forEach(param => {
                        hostMapping = hostMapping.replace(`%%${param}%%`, task.arguments[param]);
                    });
                    return hostMapping;
                }
            } catch (e) {
                console.error('Failed to parse deployment config', e);
                return null;
            }
        }
        return null;
    }

    public getJobMenu(rootTask: Task): DropdownMenuSettings {
        let menuItems: MenuItem[] = [];
        menuItems.push({
            title: 'Resubmit',
            iconName: 'fa-refresh',
            action: () => this.resubmitTask(rootTask)
        });

        // TODO (alexander): Uncomment 'Resubmit Failed' once API support is fixed.
        // if (TaskStatus.Failed === rootTask.status) {
        //     menuItems.push({
        //         title: 'Resubmit Failed',
        //         iconName: 'fa-refresh',
        //         action: () => this.resubmitTask(rootTask, true)
        //     });
        // }

        if ([TaskStatus.Cancelled, TaskStatus.Failed, TaskStatus.Success].indexOf(rootTask.status) === -1) {
            menuItems.push({
                title: 'Cancel',
                iconName: 'fa-remove',
                action: () => this.cancelTask(rootTask.id, rootTask.template.name)
            });
        }

        let actionMenu: DropdownMenuSettings = new DropdownMenuSettings(menuItems);
        actionMenu.icon = 'fa-ellipsis-v';
        return actionMenu;
    }

    public getWorkflowControlMenu(rootTask: Task): DropdownMenuSettings {
        let menuItems: MenuItem[] = [];
        menuItems.push({
            title: 'Retry',
            iconName: 'fa-refresh',
            action: () => this.resubmitTask(rootTask)
        });

        if ([TaskStatus.Cancelled, TaskStatus.Failed, TaskStatus.Success].indexOf(rootTask.status) === -1) {
            menuItems.push({
                title: 'Cancel',
                iconName: 'fa-remove',
                action: () => this.cancelTask(rootTask.id, rootTask.template.name)
            });
        }

        let actionMenu: DropdownMenuSettings = new DropdownMenuSettings(menuItems);
        actionMenu.icon = 'fa-ellipsis-v';
        return actionMenu;
    }

    public getArtifactMenuItems(rootTask: Task, step: JobTreeNode): MenuItem[] {
        let artifacts: TaskArtifact[] = this.getTaskArtifacts(step.value);
        let items: MenuItem[] = [];

        if (artifacts) {
            artifacts.forEach(item => {
                items.push({
                    title: 'Download artifact ' + item.name,
                    iconName: 'fa-download',
                    action: async () => {
                        window.location.href = await this.artifactsService.getArtifactDownloadUrlByName(step.value.id, item.name);
                    }
                });
                if (item.browsable) {
                    items.push({
                        title: 'Browse artifact ' + item.name,
                        iconName: 'fa-folder-open',
                        action: async () => this.router.navigate([`/app/jobs/job-details/${rootTask.task_id}`, {
                            tab: 'workflow',
                            browseStepArtifact: encodeURIComponent(await this.artifactsService.getArtifactDownloadUrlByName(step.value.id, item.name))
                        }])
                    });
                }
            });
        }
        return items;
    }

    public resubmitTask(rootTask, runPartial = false): any {
        this.taskService.getTask(rootTask.id).subscribe(task => {
            if (!runPartial) {
                let args = {
                    arguments: task.arguments,
                };
                if (task.hasOwnProperty('template_id')) {
                    args['template_id'] = task.template_id;
                } else {
                    args['template'] = task.template;
                }
                this.taskService.launchTask(args).subscribe(newTask => {
                    this.updateJobsList.emit({});
                    let jobUrl = this.locationStrategy.prepareExternalUrl(`app/timeline/jobs/${newTask.id}`);
                    this.notificationsService.success(
                        this.domSanitizer.bypassSecurityTrustHtml(`The job <a href="${jobUrl}">${newTask.template.name}</a> has been started.`));
                    return true;
                });
            } else {
                this.launchPanelService.openPanel(task.commit, task, true, null, true);
            }
        });
    }

    public cancelTask(id: string, name: string): any {
        this.taskService.cancelTask(id).subscribe(() => {
            this.updateJobsList.emit({});
            this.notificationsService.success(`The job ${name} has been cancelled. Please wait few seconds until job status is updated.`);
        });
    }
}

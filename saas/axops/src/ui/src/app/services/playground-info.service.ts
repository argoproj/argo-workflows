import { Injectable } from '@angular/core';
import { DomSanitizer, SafeResourceUrl } from '@angular/platform-browser';
import { Observable, Observer, Subscription } from 'rxjs';

import { Project, Task, TaskStatus } from '../model';
import { ProjectService } from './project.service';
import { ViewPreferencesService } from './view-preferences.service';
import { TaskService } from './task.service';
import { DeploymentsService } from './deployments.service';
import { JobTreeNode } from '../common/workflow-tree/workflow-tree.view-models';

const CI_WORKFLOW = {
    name: 'CI Workflow',
    repo: 'https://github.com/Applatix/ci-workflow.git',
    type: 'job',
    code: 'ci_workflow',
    videoUrl: 'https://player.vimeo.com/video/207373381',
    states: [
        { completedSteps: [], desc: 'Checking out the code from the specified repo' },
        { completedSteps: ['checkout'], desc: 'Building the checked out code' },
        { completedSteps: ['build'], desc: 'Running tests' },
        {
            completedSteps: ['test1', 'test2', 'test3', 'test4'],
            desc: 'An approval request is sent to "{required_approvals}". Please check your email and approve to complete the remaining workflow steps' },
        { completedSteps: ['approval'], desc: 'Releasing artifacts' },
        { completedSteps: ['release'], desc: 'You can now browse and download the released artifacts' },
    ]
};

const SELENIUM_TEST = {
    repo: 'https://github.com/Applatix/playground-appstore.git',
    name: 'Selenium testing',
    type: 'job',
    code: 'selenium',
    videoUrl: 'https://player.vimeo.com/video/207373951',
    states: [
        { completedSteps: [], desc: 'Checking out the code from the specified repo' },
        { completedSteps: ['checkout'], desc: 'Running tests' },
        { completedSteps: ['test_with_video'], desc: 'Expand the workflow to view the test report for the E2E_TEST step.' },
    ]
};
const WEB_APP = {
    repo: 'https://github.com/Applatix/playground-appstore.git',
    name: 'Web App',
    type: 'deployment',
    code: 'webapp',
    videoUrl: 'https://player.vimeo.com/video/207365190',
    states: [
        { completedSteps: [], desc: 'Checking out the code from the specified repo' },
        { completedSteps: ['checkout'], desc: 'Deploying MongoDb' },
        { completedSteps: ['deploy-mongo'], desc: 'Inserting data to MongoDb' },
        { completedSteps: ['insert-data'], desc: 'Deploying application' },
        { completedSteps: ['deploy-mlb'], desc: 'You can access the deployed application by clicking on the Access URL link.' },
    ]
};
const KUBERNETES = {
    repo: 'https://github.com/Applatix/playground-appstore.git',
    name: 'Kubernetes Operational View',
    type: 'deployment',
    code: 'kubernetes',
    videoUrl: 'https://player.vimeo.com/video/207365190',
    states: [
        { completedSteps: [], desc: 'Deploying Redis DB' },
        { completedSteps: ['kube-ops-view-redis-start'], desc: 'Deploying application' },
        { completedSteps: ['kube-ops-view-main-run'], desc: 'You can access the deployed application by clicking on the Access URL link.' },
    ]
};

const PROJECTS_INFO = [ CI_WORKFLOW, SELENIUM_TEST, WEB_APP, KUBERNETES ];

export interface PlaygroundProjects {
    ciWorkflowProject: Project;
    seleniumTestProject: Project;
    webAppProject: Project;
    kubernetesProject: Project;
}

export interface PlaygroundTaskInfo {
    code: string;
    jobId: string;
    backUrl: string;
    stateDescription: string;
    name: string;
    videoUrl: SafeResourceUrl;
}

@Injectable()
export class PlaygroundInfoService {

    constructor(
        private projectService: ProjectService,
        private viewPreferencesService: ViewPreferencesService,
        private taskService: TaskService,
        private deploymentsService: DeploymentsService,
        private domSanitizer: DomSanitizer) {
    }

    public loadPlaygroundProjects(): Promise<PlaygroundProjects> {
        return this.projectService.getProjects().then(projects => ({
            ciWorkflowProject: this.getProject(projects, CI_WORKFLOW),
            seleniumTestProject: this.getProject(projects, SELENIUM_TEST),
            webAppProject: this.getProject(projects, WEB_APP),
            kubernetesProject: this.getProject(projects, KUBERNETES),
        }));
    }

    public startPlaygroundTask(project: Project, task: Task) {
        this.viewPreferencesService.updateViewPreferences(viewPreferences => {
            viewPreferences.playgroundTask = {
                projectId: project ? project.id : null,
                jobId: task ? task.id : null,
            };
        });
    }

    public getPlaygroundTaskInfo(): Observable<PlaygroundTaskInfo> {
        return Observable.create((observer: Observer<PlaygroundTaskInfo>) => {
            let jobUpdatesSubscription: Subscription = null;

            let ensureJobUpdatesUnsubscribed = () => {
                if (jobUpdatesSubscription !== null) {
                    jobUpdatesSubscription.unsubscribe();
                    jobUpdatesSubscription = null;
                }
            };

            let prefSubscription = Observable.merge(Observable.fromPromise(
                    this.viewPreferencesService.getViewPreferences()), this.viewPreferencesService.onPreferencesUpdated.asObservable()).subscribe(preferences => {

                if (preferences.playgroundTask) {
                    if (preferences.playgroundTask.projectId && preferences.playgroundTask.jobId) {
                        this.projectService.getProjectById(preferences.playgroundTask.projectId).then(project => {
                            ensureJobUpdatesUnsubscribed();
                            jobUpdatesSubscription = this.taskService.getTaskUpdates(preferences.playgroundTask.jobId).subscribe(task => {
                                this.getTaskInfo(task, project).then(info => observer.next(info));
                            });
                        });
                    } else {
                        observer.next({
                            code: 'cashboard',
                            jobId: '',
                            backUrl: '/app/cashboard',
                            stateDescription: 'Monitor your cloud cost and resource usage',
                            name: 'Cashboard',
                            videoUrl: null
                        });
                    }
                } else {
                    observer.next(null);
                }
            });

            return () => {
                prefSubscription.unsubscribe();
                ensureJobUpdatesUnsubscribed();
            };
        });

    }

    private getTaskInfo(task: Task, project: Project): Promise<PlaygroundTaskInfo> {
        let info = PROJECTS_INFO.find(item => item.repo === project.repo && item.name === project.name);
        if (!info) {
            return null;
        }

        let backUrl = Promise.resolve(`/app/timeline/jobs/${task.id}`);
        if (info.type === 'deployment' && task.status === TaskStatus.Success) {
            let deploymentSteps = task.children.filter(item => item.template.type === 'deployment');
            if (deploymentSteps.length > 0) {
                backUrl = this.deploymentsService.getDeploymentById(deploymentSteps[0].id).toPromise().then(deployment => `/app/applications/details/${deployment.app_generation}`);
            }
        }
        let stateDescription = this.getStateDescription(task, info.states);
        return backUrl.then(url => {
            return {
                jobId: task.id,
                code: info.code,
                backUrl: url,
                stateDescription,
                videoUrl: info.videoUrl ? this.domSanitizer.bypassSecurityTrustResourceUrl(info.videoUrl) : null,
                name: info.name,
            };
        });
    }

    private getProject(projects: Project[], config: {name: string, repo: string}) {
        return projects.find(project => project.repo === config.repo && project.name === config.name);
    }

    private getStateDescription(task: Task, states: any[]) {
        let steps = JobTreeNode.createFromTask(task).getFlattenNodes();
        let state = null;
        for (let nextState of states) {
            let completedStepsCount = nextState.completedSteps.filter(stepName =>
                steps.find(step => step.name.toLowerCase().startsWith(stepName) && step.value.status === TaskStatus.Success)
            ).length;
            if (completedStepsCount === nextState.completedSteps.length) {
                state = nextState;
            } else {
                break;
            }
        }
        let stateDescription: string = state && state.desc || '';
        for (let param in task.arguments || {}) {
            if (task.arguments.hasOwnProperty(param)) {
                stateDescription = stateDescription.replace(`{${param}}`, task.arguments[param]);
            }
        }
        return stateDescription;
    }
}

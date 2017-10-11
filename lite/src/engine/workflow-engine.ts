import { Observable } from 'rxjs';
import * as moment from 'moment';
import * as uuid from 'uuid';
import * as fs from 'fs';
import * as tar from 'tar-fs';
import { Docker } from 'node-docker-api';

import { WorkflowOrchestrator } from './workflow-orchestrator';
import { Executor, StepResult, Logger } from './common';
import * as utils from './utils';
import * as model from './model';

export class WorkflowEngine {
    private orchestrator: WorkflowOrchestrator;
    private taskResultsById = new Map<string, { task: model.Task; stepResults: { [id: string]: StepResult }}>();
    private stepResultsById = new Map<string, StepResult>();

    constructor(private executor: Executor, private logger: Logger, private docker: Docker) {
        this.orchestrator = new WorkflowOrchestrator(executor, logger, docker);
        this.orchestrator.getStepResults().subscribe(res => {
            this.stepResultsById.set(res.id, res.result);
            let taskResult = this.taskResultsById.get(res.taskId);
            let rootTask = taskResult.task;
            if (res.id === res.taskId) {
                 this.updateTaskStatus(rootTask, res.result);
             } else {
                 let childStep = rootTask.children.find(child => child.id === res.id);
                 this.updateTaskStatus(childStep, res.result);
                 taskResult.stepResults[res.id] = res.result;
             }
        });
    }

    public getServiceEvents(taskId?: string): Observable<model.TaskEvent> {
        let events = this.orchestrator.getStepResults().map(res => ({ task_id: res.taskId, id: res.id, status: res.result.status }));
        if (taskId) {
            events = events.filter(event => event.task_id === taskId);
        }
        return events;
    }

    public async launch(template: model.Template, args: {[name: string]: string}): Promise<model.Task> {
        let task = this.constructTask(template, args);
        this.taskResultsById.set(task.id, { task, stepResults: {} });
        this.orchestrator.processTask(task);
        return task;
    }

    public getTaskById(id: string) {
        let result = this.taskResultsById.get(id);
        return result && result.task || null;
    }

    public getTasks(): model.Task[] {
        return Array.from(this.taskResultsById.values()).map(item => item.task).sort((first, second) => second.create_time - first.create_time);
    }

    public getStepArtifact(id: string, artifactName: string) {
        let stepResult = this.stepResultsById.get(id);
        if (stepResult && stepResult.artifacts[artifactName]) {
            return utils.reactifyStream(tar.pack(stepResult.artifacts[artifactName]));
        }
        return null;
    }

    public getTaskArtifacts(id: string): {name: string, artifact_type: string, workflow_id: string }[] {
        let task = this.taskResultsById.get(id);
        if (task) {
            let artifacts = Object.keys(task.stepResults).map(stepId => {
                return Object.keys(task.stepResults[stepId].artifacts || {}).map(artifactName => ({
                    artifact_id: `${stepId}:${artifactName}`,
                    name: artifactName,
                    artifact_type: 'internal',
                    workflow_id: id,
                }));
            });
            return artifacts.reduce((first, second) => first.concat(second), []);
        }
        return [];
    }

    public getStepLogs(id: string): Observable<string> {
        let logs: Observable<string> = null;
        let stepResult = this.stepResultsById.get(id);
        if (stepResult) {
            if (stepResult.status === model.TaskStatus.Running && stepResult.stepId) {
                logs = this.executor.getLiveLogs(stepResult.stepId);
            } else if (stepResult.logsPath) {
                logs = utils.reactifyStringStream(fs.createReadStream(stepResult.logsPath));
            }
        }
        return logs;
    }

    private constructTask(template: model.Template, args: {[name: string]: string}): model.Task {
        let id = uuid();
        let task: model.Task = {
            id, name: template.name, template, arguments: args, launch_time: 0, create_time: moment().unix(), task_id: id, commit: {}, artifact_tags: '' };
        task.children = [];

        function addFixtureTasks(step: model.WorkflowStep) {
            for (let fixtureGroup of (step.template.fixtures || [])) {
                Object.keys(fixtureGroup).forEach(fixtureName => {
                    let fixture = fixtureGroup[fixtureName];
                    fixture['id'] = uuid();
                    let fixtureTask = { id: fixture['id'], template: fixture.template, launch_time: 0, create_time: moment().unix(), status: model.TaskStatus.Init };
                    task.children.push(fixtureTask);
                });
            }
        }

        addFixtureTasks(task);
        let childGroups = (task.template.steps || []).slice();
        while (childGroups.length > 0) {
            let group = childGroups.pop();
            Object.keys(group).forEach(stepName => {
                let step = group[stepName];
                step.id = uuid();
                let stepTask: model.Task = { id: step.id, template: step.template, launch_time: 0, create_time: moment().unix(), status: model.TaskStatus.Init };
                task.children.push(stepTask);
                if (step.template.steps) {
                    childGroups = childGroups.concat(step.template.steps);
                }
                addFixtureTasks(step);
            });
        }
        return task;
    }

    private updateTaskStatus(task: model.Task, stepResult: StepResult) {
        task.status = stepResult.status;
        if (task.status === stepResult.status) {
            switch (stepResult.status) {
                case model.TaskStatus.Running:
                    task.launch_time = moment().unix();
                    break;
                default:
                    task.run_time = moment().unix() - (task.launch_time || task.create_time);
                    break;
            }
            let message = '';
            if (stepResult.internalError) {
                message = stepResult.internalError instanceof Error ? stepResult.internalError.message : JSON.stringify(stepResult.internalError);
            }
            task['status_detail'] = {
                code: this.getStatusCode(stepResult.status),
                message,
            };
        }
    }

    private getStatusCode(status: model.TaskStatus): string {
        switch (status) {
            case model.TaskStatus.Skipped: return 'Skipped';
            case model.TaskStatus.Cancelled: return 'Cancelled';
            case model.TaskStatus.Failed: return 'Failed';
            case model.TaskStatus.Success: return 'Success';
            case model.TaskStatus.Waiting: return 'Waiting';
            case model.TaskStatus.Running: return 'Running';
            case model.TaskStatus.Canceling: return 'Canceling';
            case model.TaskStatus.Init: return 'Init';
            default: return '';
        }
    }
}

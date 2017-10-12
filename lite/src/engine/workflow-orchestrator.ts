import { Subject, Observable, Subscription } from 'rxjs';
import { Docker } from 'node-docker-api';
import * as deepcopy from 'deepcopy';

import * as model from './model';
import * as utils from './utils';
import { Executor, StepResult, WorkflowContext, StepInput, Logger } from './common';

interface LaunchedFixture { name: string; networkName: string; subscription: Subscription; }

export class WorkflowOrchestrator {
    private readonly stepResultsQueue = new Subject<{id: string, taskId: string, result: StepResult}>();
    private readonly tasksProcessingQueue = new Subject<model.Task>();

    constructor(private executor: Executor, private logger: Logger, private docker: Docker) {
        this.tasksProcessingQueue.subscribe(async task => {
            try {
                this.logger.debug(`New task received: id: '${task.id}'`);
                await this.processStep(
                    task.id,
                    {id: task.id, template: task.template, arguments: task.arguments},
                    {workflow: null, results: {}, input: null},
                    {parameters: task.arguments, artifacts: {}, fixtures: {}});
                this.logger.debug(`Task has been processed: id: '${task.id}'`);
            } catch (e) {
                this.logger.error('An internal error during task processing', e);
                this.stepResultsQueue.next({id: task.id, taskId: task.id, result: { status: model.TaskStatus.Failed, internalError: e }});
            }
        });
    }

    public getStepResults(): Observable<{id: string, taskId: string, result: StepResult}> {
        return this.stepResultsQueue;
    }

    public processTask(task: model.Task) {
        this.tasksProcessingQueue.next(task);
    }

    private processStep(taskId: string, step: model.WorkflowStep, parentContext: WorkflowContext, input: StepInput): Promise<StepResult> {
        try {
            this.populateDefaultArgs(step, input);
            switch (step.template.type) {
                case 'workflow':
                    return this.processWorkflow(taskId, step, parentContext, input);
                case 'container':
                    return this.processContainer(taskId, step, parentContext, input);
                default:
                    throw new Error(`Type ${step.template.type} is not supported`);
            }
        } catch (e) {
            this.stepResultsQueue.next({ id: step.id, taskId, result: { status: model.TaskStatus.Failed, internalError: e } });
            throw e;
        }
    }

    private populateDefaultArgs(step: model.WorkflowStep, input: StepInput) {
        input.parameters = input.parameters || {};
        Object.keys(step.template.inputs && step.template.inputs.parameters || {}).forEach(paramKey => {
            let defaultVal = step.template.inputs.parameters[paramKey] && step.template.inputs.parameters[paramKey].default;
            if (defaultVal && !input.parameters.hasOwnProperty(`parameters.${paramKey}`)) {
                input.parameters[`parameters.${paramKey}`] = defaultVal;
            }
        });
    }

    private async processWorkflow(taskId: string, workflow: model.WorkflowStep, parentContext: WorkflowContext, input: StepInput): Promise<StepResult> {
        input = this.processStepInput(workflow, parentContext, input);
        let fixtures: LaunchedFixture[] = [];
        let networkId;
        try {
            let result: StepResult = { status: model.TaskStatus.Running, artifacts: {}, statusCode: model.TASK_STATUS_CODES.TASK_WAITING };
            this.stepResultsQueue.next({ result, id: workflow.id, taskId });

            if (workflow.template.fixtures && workflow.template.fixtures.length > 0) {
                networkId = await this.executor.createNetwork(workflow.id);
                input.networkId = networkId;
            }

            if (workflow.template.fixtures) {
                result.statusCode = model.TASK_STATUS_CODES.SETUP_FIXTURE;
                this.stepResultsQueue.next({ result, id: workflow.id, taskId });
            }
            fixtures = await this.startFixtures(taskId, workflow, parentContext, input);

            result.statusCode = model.TASK_STATUS_CODES.CONTAINER_RUNNING;
            this.stepResultsQueue.next({ result, id: workflow.id, taskId });

            let context: WorkflowContext = { workflow, results: {}, input };
            for (let parallelStepsGroup of workflow.template.steps) {
                let results = await Promise.all(
                    Object.keys(parallelStepsGroup)
                        .map(stepName => {
                            this.logger.debug(`Starting step '${stepName}': id: '${parallelStepsGroup[stepName].id}'`);
                            return this.processStep(taskId, parallelStepsGroup[stepName], context, input).then(res => {
                                let stepResult = Object.assign(res, { name: stepName });
                                this.logger.debug(`Step '${stepName}' has been completed`);
                                return stepResult;
                            });
                        }),
                );

                for (let stepResult of results) {
                    context.results[stepResult.name] = stepResult;

                    Object.keys(stepResult.artifacts || {}).forEach(artifactName => {
                        let matchingArtifactName = Object.keys((workflow.template.outputs || {}).artifacts || {}).find(key =>
                            workflow.template.outputs.artifacts[key].from === `%%steps.${stepResult.name}.outputs.artifacts.${artifactName}%%`);
                        if (matchingArtifactName) {
                            result.artifacts[matchingArtifactName] = stepResult.artifacts[artifactName];
                        }
                    });
                    if (stepResult.status === model.TaskStatus.Failed) {
                        result.status = model.TaskStatus.Failed;
                        result.statusCode = model.TASK_STATUS_CODES.TASK_FAILED;
                        this.stepResultsQueue.next({ result, id: workflow.id, taskId });
                    }
                }
                if (result.status === model.TaskStatus.Failed) {
                    break;
                }
            }
            if (result.status === model.TaskStatus.Running) {
                result.status = model.TaskStatus.Success;
                result.statusCode = model.TASK_STATUS_CODES.TASK_SUCCEED;
                this.stepResultsQueue.next({ result, id: workflow.id, taskId });
            }
            return result;
        } finally {
            for (let fixture of fixtures) {
                await fixture.subscription.unsubscribe();
            }
            if (networkId) {
                await this.deleteNetworkSafe(networkId);
            }
        }
    }

    private async deleteNetworkSafe(networkId: string) {
        await utils.executeSafe(async () => {
            try {
                this.logger.debug(`Deleting network: id: '${networkId}';`);
                await this.executor.deleteNetwork(networkId);
                this.logger.debug(`Network was successfully deleted: '${networkId}'`);
            } catch (e) {
                this.logger.debug(`Failed to delete network: id: '${networkId}'`);
                throw e;
            }
        }, 5, 300);
    }

    private async startFixtures(
            taskId: string, workflow: model.WorkflowStep, parentContext: WorkflowContext, input: StepInput): Promise<LaunchedFixture[]> {
        let results: LaunchedFixture[] = [];
        for (let group of workflow.template.fixtures || []) {
            let groupResultPromises = Object.keys(group).map(fixtureName => {
                let fixture = group[fixtureName];
                this.logger.debug(`Starting fixture '${fixtureName}': id: '${fixture['id']}'`);
                return new Promise((resolve, reject) => {
                    let started = false;
                    let subscription = this.launchContainer(fixture, parentContext, input).subscribe(fixtureResult => {
                        this.stepResultsQueue.next({ id: fixture['id'], taskId, result: fixtureResult });
                        if (!started && fixtureResult.status === model.TaskStatus.Running) {
                            if (!fixtureResult.networkName) {
                                reject(new Error(`Fixture '${fixtureName}' had been started by network name/IP is unknown: ${JSON.stringify(fixtureResult)}`));
                            }
                            this.logger.debug(`Fixture '${fixtureName}' has been started and available at '${fixtureResult.networkName}'`);
                            resolve({ name: fixtureName, subscription, networkName: fixtureResult.networkName });
                            started = true;
                        } else if (!started && fixtureResult.status === model.TaskStatus.Failed) {
                            reject(new Error(`Unable to start fixture ${fixtureName}: ${JSON.stringify(fixtureResult)}`));
                        }
                    });
                });
            });
            let groupResults = <LaunchedFixture[]> await Promise.all(groupResultPromises);
            groupResults.forEach(res => input.fixtures[res.name] = res.networkName);
            results = results.concat(groupResults);
        }
        return results;
    }

    private processContainer(taskId: string, container: model.WorkflowStep, parentContext: WorkflowContext, input: StepInput): Promise<StepResult> {
        return this.launchContainer(container, parentContext, input)
            .do(result => this.stepResultsQueue.next({id: container.id, taskId, result}))
            .last().toPromise();
    }

    private getDockerParams(container: model.WorkflowStep) {
        try {
            if (container.template.annotations['ax_ea_docker_enable']) {
                let params = JSON.parse(container.template.annotations['ax_ea_docker_enable']);
                return {
                    graphStorageSize: params['graph-storage-size'],
                    cpuCores: params['cpu_cores'],
                    memMib: params['mem_mib'],
                };
            }
        } catch (e) {
            return null;
        }
    }

    private launchContainer(container: model.WorkflowStep, parentContext: WorkflowContext, input: StepInput): Observable<StepResult> {
        container = deepcopy(container);
        return Observable.fromPromise((async () => {
            input = this.processStepInput(container, parentContext, input);
            if (!container.template.command) {
                let imageEntryPoint = await this.getImageEntryPoint(container.template.image);
                container.template.command = imageEntryPoint;
                container.template.args = [];
            }
            container.template.command = container.template.command.map(item => this.substituteInputParams(item, input));
            container.template.args = (container.template.args || []).map(item => this.substituteInputParams(item, input));
            container.template.image = this.substituteInputParams(container.template.image, input);
        })()).first().flatMap(() => this.executor.executeContainerStep(container, parentContext, {
            artifacts: input.artifacts,
            networkId: input.networkId,
            dockerParams: this.getDockerParams(container),
        }));
    }

    private async getImageEntryPoint(imageUrl: string) {
        await utils.ensureImageExists(this.docker, imageUrl);
        let status = await this.docker.image.get(imageUrl).status();
        return status.data['ContainerConfig'].Entrypoint;
    }

    private processStepInput(step: model.WorkflowStep, parentContext: WorkflowContext, input: StepInput): StepInput {
        let parameters = {};
        Object.keys(step.arguments || {}).forEach(key => {
            parameters[key] = parentContext.input ? this.substituteInputParams(step.arguments[key], parentContext.input) : step.arguments[key];
        });
        let artifacts = {};
        Object.keys(step.arguments || {}).filter(name => name.startsWith('artifacts.')).forEach(name => {
            let stepsArtifactMatch = step.arguments[name].match(/%%steps\.([^\.]*)\.outputs\.artifacts\.([^%.]*)%%/);
            let inputsArtifactsMatch = step.arguments[name].match(/%%inputs\.artifacts\.([^%.]*)%%/);
            if (stepsArtifactMatch) {
                let [, stepName, artifactName] = stepsArtifactMatch;
                let stepResult = parentContext.results[stepName];
                if (!stepResult) {
                    throw new Error(`Step requires output artifact of step '${stepName}', but step result is not available`);
                }
                artifacts[artifactName] = (stepResult.artifacts || {})[artifactName];
            } else if (inputsArtifactsMatch) {
                let [, artifactName] = inputsArtifactsMatch;
                let artifact = input.artifacts[artifactName];
                if (!artifact) {
                    throw new Error(`Step requires input artifact'${artifactName}', but artifact is not available`);
                }
                artifacts[artifactName] = artifact;
            } else {
                throw new Error(`Unable to parse artifacts input: '${step.arguments[name]}'`);
            }
        });
        return { parameters, artifacts, networkId: input.networkId, fixtures: input.fixtures };
    }

    private substituteInputParams(src: string, input: StepInput) {
        Object.keys(input.parameters).filter(key => key.startsWith('parameters.')).forEach(key => {
            src = src.replace(new RegExp(`%%inputs.${key}%%`, 'g'), input.parameters[key]);
        });
        Object.keys(input.fixtures).forEach(key => {
            src = src.replace(new RegExp(`%%fixtures.${key}.ip%%`, 'g'), input.fixtures[key]);
        });
        return src;
    }
}

import { Observable, Observer } from 'rxjs';
import * as api from 'kubernetes-client';
import * as path from 'path';
import * as shell from 'shelljs';
import * as shellEscape from 'shell-escape';
import * as fs from 'fs';

import * as model from './model';
import { Executor, StepResult, WorkflowContext, Logger, ContainerStepInput } from './common';
import * as utils from './utils';

interface PodInfo { name: string; podIP: string; status: 'Pending' | 'Running' | 'Succeeded' | 'Failed'; };

export class KubernetesExecutor implements Executor {

    public static fromConfigFile(logger: Logger, configPath: string, namespace: string, version = 'v1') {
        let config = Object.assign({}, api.config.fromKubeconfig(api.config.loadKubeconfig(configPath)), {namespace, version });
        return new KubernetesExecutor(logger, configPath, config);
    }

    public static inCluster(logger: Logger) {
        let config = Object.assign({}, api.config.getInCluster());
        return new KubernetesExecutor(logger, '', config);
    }

    private core: api.Core;
    private podUpdates: Observable<PodInfo>;

    private constructor(private logger: Logger, private configPath: string, private config: any) {
        this.core = new api.Core(Object.assign(config, {promises: true}));

        this.podUpdates = utils.
            reactifyJsonStream(this.core.ns.pods.getStream({ qs: { watch: true } })).
            map(item => item.object).
            map(item => {
                let name = item.metadata.name;
                let status: 'Pending' | 'Running' | 'Succeeded' | 'Failed';
                switch (item.status.phase) {
                    case 'Failed':
                    case 'Pending':
                    case 'Succeeded':
                        status = item.status.phase;
                        break;
                    case 'Running':
                        let stepExitCode = this.getContainerExitCode(item, 'step');
                        let dockerExitCode = this.getContainerExitCode(item, 'docker');
                        if ((stepExitCode !== null && stepExitCode !== 0) || (dockerExitCode !== null && dockerExitCode !== 0)) {
                            status = 'Failed';
                        } if (stepExitCode === 0) {
                            status = 'Succeeded';
                        } else {
                            status = 'Running';
                        }
                        break;
                    default:
                        status = 'Failed';
                        break;
                }
                return { name, status, podIP: item.status.podIP || null };
            }).
            distinct(item => `${item.name} - ${item.status} - ${item.podIP}`).
            do(item => this.logger.debug(`Pod '${item.name}' has been updated: status: ${item.status}; podIp: ${item.podIP}`)).
            share();
    }

    public async createNetwork(name: string): Promise<string> {
        // do nothing, pods are running in same network
        return name;
    }

    public async deleteNetwork(name: string): Promise<any> {
        // do nothing, pods are running in same network
    }

    public executeContainerStep(step: model.WorkflowStep, context: WorkflowContext, input: ContainerStepInput): Observable<StepResult> {
        return new Observable<StepResult>((observer: Observer<StepResult>) => {
            let stepPod = null;

            let cleanUp = async () => {
                if (stepPod) {
                    try {
                        await this.core.ns.pods.delete({ name: stepPod.metadata.name });
                        stepPod = null;
                    } catch (e) {
                        stepPod = null;
                    }
                }
            };
            let result: StepResult = { status: model.TaskStatus.Waiting };

            function notify(update: StepResult) {
                observer.next(Object.assign(result, update));
            }

            let execute = async () => {

                try {
                    let tempDir = path.join(shell.tempdir(), 'argo', step.id);

                    notify({ status: model.TaskStatus.Waiting });
                    stepPod = await this.createPod(step, input);

                    let startedPod = await this.podUpdates.filter(pod => pod.name === step.id && pod.status !== 'Pending').first().toPromise();

                    await this.uploadArtifacts(step, stepPod, input);

                    await this.kubeCtlExec(stepPod, ['echo done > /__argo/artifacts_in']);

                    notify({ status: model.TaskStatus.Running, stepId: stepPod.metadata.name, networkName: startedPod.podIP });

                    this.logger.debug(`Running user script for for step id ${step.id}`);
                    let stepIsDone = false;
                    do {
                        let res = await this.kubeCtlExec(stepPod, ['cat /__argo/step_done'], false);
                        stepIsDone = res.code === 0 && (res.stdout || '').trim() === 'done';
                    } while (!stepIsDone);
                    this.logger.debug(`User script for for step id ${step.id} has been completed`);

                    let artifactsMap = await this.downloadArtifacts(step, stepPod, input, tempDir);

                    await this.kubeCtlExec(stepPod, ['echo done > /__argo/artifacts_out']);
                    let completedPod = await this.podUpdates.filter(pod => pod.name === step.id && this.isPodCompeleted(pod)).first().toPromise();

                    let logLines = await this.getLiveLogs(stepPod.metadata.name).toArray().toPromise();
                    let logsPath = path.join(tempDir, 'logs');
                    fs.writeFileSync(logsPath, logLines.join(''));
                    notify({ logsPath });

                    notify({
                        status: completedPod.status === 'Succeeded' ? model.TaskStatus.Success : model.TaskStatus.Failed,
                        artifacts: artifactsMap,
                    });
                } catch (e) {
                    notify({ status: model.TaskStatus.Failed, internalError: e });
                } finally {
                    await cleanUp();
                    observer.complete();
                }
            };

            execute();
            return cleanUp;
        });
    }

    public getLiveLogs(containerId: string): Observable<string> {
        return utils.reactifyStringStream(this.core.ns.po(containerId).log.getStream({ qs: { follow: true, container: 'step' } }));
    }

    private getContainerExitCode(pod, containerName: string) {
        let container = pod.status.containerStatuses.find(item => item.name === containerName);
        if (container && container.state.terminated) {
            return container.state.terminated.exitCode;
        }
        return null;
    }

    private createPod(step: model.WorkflowStep, input: ContainerStepInput) {
        let volumes = [];
        let env = [];
        if (input.dockerParams) {
            env.push({name: 'DOCKER_HOST', value: 'tcp://localhost:2375'});
        }
        let containers = [{
            name: 'step',
            image: step.template.image,
            command: ['sh', '-c'],
            env,
            args: [
                `mkdir -p /__argo;
                until [ -f /__argo/artifacts_in ]; do sleep 1; done;
                ${input.dockerParams ? 'until docker info; do sleep 1; done;' : ''}
                ${shellEscape(step.template.command.concat(step.template.args))};script_exit_code=$?;
                echo done > /__argo/step_done;
                until [ -f /__argo/artifacts_out ]; do sleep 1; done;
                exit $script_exit_code`,
            ],
            resources: step.template.resources && {
                requests: {
                    memory: step.template.resources.mem_mib && `${step.template.resources.mem_mib}Mi`,
                    cpu: step.template.resources.cpu_cores && `${step.template.resources.cpu_cores}m`,
                },
            },
            securityContext: null,
            volumeMounts: [],
        }];

        if (input.dockerParams) {
            containers.push({
                name: 'docker',
                image: 'docker:1.12.6-dind',
                command: null,
                args: null,
                env: null,
                resources: {
                    requests: {
                        memory: input.dockerParams.memMib && `${input.dockerParams.memMib}Mi`,
                        cpu: input.dockerParams.cpuCores && `${input.dockerParams.cpuCores}m`,
                    },
                },
                securityContext: { privileged: true },
                volumeMounts: [{ name: 'docker-graph-storage', mountPath: '/var/lib/docker' }],
            });
            volumes.push({name: 'docker-graph-storage', emptyDir: {}});
        }

        return this.core.ns.pods.post({body: {
            apiVersion: 'v1',
            kind: 'Pod',
            metadata: { name: step.id },
            spec: { containers, restartPolicy: 'Never', volumes },
        }});
    }

    private uploadArtifacts(step: model.WorkflowStep, stepPod, input: ContainerStepInput) {
        return Promise.all(Object.keys(step.template.inputs && step.template.inputs.artifacts || {}).map(async artifactName => {
            let inputArtifactPath = input.artifacts[artifactName];
            let artifact = step.template.inputs.artifacts[artifactName];
            this.logger.debug(`Uploading artifacts to '${artifact.path}' for step id ${step.id}`);
            await this.runKubeCtl(['cp', inputArtifactPath, `${stepPod.metadata.name}:/__argo/`, '-c', 'step'], true);
            await this.kubeCtlExec(stepPod, [`mkdir -p ${path.dirname(artifact.path)} && mv /__argo/${path.basename(inputArtifactPath)} ${artifact.path}`], true);
            this.logger.debug(`Successfully uploaded artifacts to '${artifact.path}' for step id ${step.id}`);
        }));
    }

    private async downloadArtifacts(step: model.WorkflowStep, stepPod, input: ContainerStepInput, tempDir: string) {
        let artifactsDir = path.join(tempDir, 'artifacts');
        shell.mkdir('-p', artifactsDir);

        let artifacts = step.template.outputs && step.template.outputs.artifacts && await Promise.all(Object.keys(step.template.outputs.artifacts).map(async key => {
            let artifact = step.template.outputs.artifacts[key];
            let artifactPath = path.join(artifactsDir, key);
            this.logger.debug(`Downloading artifacts from '${artifact.path}' for step id ${step.id}`);
            await this.runKubeCtl(['cp', `${stepPod.metadata.name}:${artifact.path}`, artifactPath, '-c', 'step']);
            this.logger.debug(`Successfully downloaded artifacts from '${artifact.path}' for step id ${step.id}`);
            return { name: key, artifactPath };
        })) || [];

        let artifactsMap = {};
        artifacts.forEach(item => artifactsMap[item.name] = item.artifactPath);
        return artifactsMap;
    }

    private kubeCtlExec(stepPod: any, cmd: string[], rejectOnFail = true) {
        return this.runKubeCtl(['exec', `${stepPod.metadata.name}`, '--', 'sh', '-c'].concat(cmd), rejectOnFail);
    }

    private runKubeCtl(cmd: string[], rejectOnFail = true) {
        let args = ['kubectl'];
        if (this.config) {
            args.push(`--kubeconfig=${this.configPath}`);
        }
        return utils.exec(args.concat(cmd), rejectOnFail);
    }

    private isPodCompeleted(pod) {
        return ['Succeeded', 'Failed', 'Unknown'].indexOf(pod.status) > -1;
    }
}

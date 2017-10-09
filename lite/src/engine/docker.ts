import * as shell from 'shelljs';
import * as path from 'path';
import * as fs from 'fs';
import { Observable, Observer } from 'rxjs';
import { Docker } from 'node-docker-api';

import * as model from './model';
import * as utils from './utils';
import { Executor, StepResult, WorkflowContext, Logger, ContainerStepInput } from './common';

export class DockerExecutor implements Executor {

    private docker: Docker;
    private emptyDir: string;

    constructor(private logger: Logger, private socketPath = '/var/run/docker.sock') {
        this.docker = new Docker({ socketPath });
        this.emptyDir = path.join(shell.tempdir(), 'empty');
        shell.mkdir('-p', this.emptyDir);
    }

    public async createNetwork(name: string): Promise<string> {
        this.logger.debug(`Creating new network: name: '${name}';`);
        let res = await this.docker.network.create({ name });
        let networkId = res.id.toString();
        await utils.execute(async () => {
            let networks = await this.docker.network.list();
            if (networks.findIndex(item => item.id === networkId) === -1 ) {
                throw new Error('Docker failed to create new network');
            }
        }, 100, 5);
        this.logger.debug(`Network has been created: id: '${networkId}' name: '${name}'`);
        return networkId;
    }

    public async deleteNetwork(id: string): Promise<any> {
        let network = this.docker.network.get(id);
        await network.remove();
    }

    public executeContainerStep(step: model.WorkflowStep, context: WorkflowContext, input: ContainerStepInput): Observable<StepResult> {
        return new Observable<StepResult>((observer: Observer<StepResult>) => {
            let container = null;
            let result: StepResult = { status: model.TaskStatus.Waiting };

            function notify(update: StepResult) {
                observer.next(Object.assign(result, update));
            }

            let cleanUpContainer = async () => {
                if (container) {
                    await this.removeContainerSafe(container);
                    container = null;
                }
            };

            let execute = async () => {
                try {
                    await this.ensureImageExists(step.template.image);

                    container = await this.createContainer(step, input);

                    if (input.networkId) {
                        await this.docker.network.get(input.networkId).connect({ container: container.id, endpointConfig: { aliases: [ step.id ] }});
                    }

                    let tempDir = path.join(shell.tempdir(), 'argo', step.id);
                    let artifactsDir = path.join(tempDir, 'artifacts');
                    shell.mkdir('-p', artifactsDir);

                    await Promise.all(Object.keys((step.template.inputs || {}).artifacts || {}).map(async artifactName => {
                        let inputArtifactPath = input.artifacts[artifactName];
                        let artifact = step.template.inputs.artifacts[artifactName];
                        await this.dockerMakeDir(container, artifact.path);
                        await utils.exec(['docker', 'cp', inputArtifactPath, `${container.id}:${artifact.path}`], false);
                    }));

                    await container.start();
                    notify({ status: model.TaskStatus.Running, stepId: container.id, networkName: step.id });

                    let status = await container.wait();

                    let logLines = await this.getContainerLogs(container).toArray().toPromise();
                    let logsPath = path.join(tempDir, 'logs');
                    fs.writeFileSync(logsPath, logLines.join(''));

                    notify({ logsPath });

                    let artifacts = step.template.outputs && step.template.outputs.artifacts && await Promise.all(Object.keys(step.template.outputs.artifacts).map(async key => {
                        let artifact = step.template.outputs.artifacts[key];
                        let artifactPath = path.join(artifactsDir, key);
                        await utils.exec(['docker', 'cp', `${container.id}:${artifact.path}`, artifactPath], false);
                        return { name: key, artifactPath };
                    })) || [];
                    let artifactsMap = {};
                    artifacts.forEach(item => artifactsMap[item.name] = item.artifactPath);
                    notify({ status: status.StatusCode === 0 ? model.TaskStatus.Success : model.TaskStatus.Failed, artifacts: artifactsMap });
                } catch (e) {
                    notify({ status: model.TaskStatus.Failed, internalError: e });
                } finally {
                    await cleanUpContainer();
                    observer.complete();
                }
            };

            execute();
            return cleanUpContainer;
        });
    }

    public getLiveLogs(containerId: string): Observable<string> {
        return this.getContainerLogs(this.docker.container.get(containerId));
    }

    private createContainer(step: model.WorkflowStep, input: ContainerStepInput) {
        let hostConfig = null;
        if (input.dockerParams) {
            hostConfig = { binds: [`${this.socketPath}:/var/run/docker.sock`] };
        }
        return this.docker.container.create({ image: step.template.image, cmd: step.template.command.concat(step.template.args), hostConfig});
    }

    private async removeContainerSafe(container): Promise<any> {
        return utils.executeSafe(async () => {
            try {
                await container.kill();
            } finally {
                await container.delete({ force: true });
            }
        }, 3, 100);
    }

    private getContainerLogs(container): Observable<string> {
        return Observable.fromPromise(container.logs({ stdout: true, stderr: true, follow: true })).flatMap(stream => utils.reactifyStringStream(stream));
    }

    private async ensureImageExists(imageUrl: string): Promise<any> {
        let res = await await this.docker.image.list({filter: imageUrl});
        if (res.length === 0) {
            await utils.exec(['docker', 'pull', imageUrl]);
        }
    }

    private async dockerMakeDir(container: any, dirPath: string) {
        let parts = path.dirname(dirPath).split('/').filter(item => !!item);
        for (let i = 0; i < parts.length; i++) {
            await utils.exec(['docker', 'cp', this.emptyDir, `${container.id}:/${parts.slice(0, i + 1).join('/')}`], false);
        }
    }
}

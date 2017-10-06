import { Observable } from 'rxjs';
import * as model from './model';

export interface WorkflowContext {
    workflow: model.WorkflowStep;
    results: {[name: string]: StepResult};
    input: StepInput;
}

export interface StepResult {
    // Executor specific step id.
    stepId?: string;
    status?: model.TaskStatus;
    // Path to file with logs. Should be available after step is completed.
    logsPath?: string;
    // Artifacts paths by name
    artifacts?: { [name: string]: string };
    internalError?: any;
    networkName?: string;
}

export interface StepInput {
    parameters: { [name: string]: string };
    artifacts: { [name: string]: string };
    fixtures: { [name: string]: string };
    networkId?: string;
}

export interface Executor {
    executeContainerStep(step: model.WorkflowStep, context: WorkflowContext, inputArtifacts: {[name: string]: string}, networkdId: string): Observable<StepResult>;
    getLiveLogs(containerId: string): Observable<string>;
    createNetwork(name: string): Promise<string>;
    deleteNetwork(name: string): Promise<any>;
}

export class Logger {
    constructor(private bunyan) {
    }

    public info(...params: any[]) {
        this.bunyan.info.call(this.bunyan, params);
    }

    public debug(...params: any[]) {
        this.bunyan.debug.call(this.bunyan, params);
    }

    public trace(...params: any[]) {
        this.bunyan.trace.call(this.bunyan, params);
    }

    public warn(...params: any[]) {
        this.bunyan.warn.call(this.bunyan, params);
    }

    public error(...params: any[]) {
        this.bunyan.error.call(this.bunyan, params);
    }
}

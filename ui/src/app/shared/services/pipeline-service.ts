import {Pipeline, PipelineList, PipelineWatchEvent} from '../../../models/pipeline';
import {LogEntry} from '../../../models/pipeline';
import {StepWatchEvent} from '../../../models/step';
import requests from './requests';

export class PipelineService {
    public listPipelines(namespace: string) {
        return requests.get(`api/v1/pipelines/${namespace}`).then(res => res.body as PipelineList);
    }

    public watchPipelines(namespace: string) {
        return requests.loadEventSource(`api/v1/stream/pipelines/${namespace}`).map(line => line && (JSON.parse(line).result as PipelineWatchEvent));
    }

    public pipelineLogs(namespace: string, name = '', stepName = '', container = 'main', grep = '', tailLines = -1) {
        const params = ['podLogOptions.follow=true'];
        if (name) {
            params.push('name=' + name);
        }
        if (stepName) {
            params.push('stepName=' + stepName);
        }
        if (container) {
            params.push('podLogOptions.container=' + container);
        }
        if (grep) {
            params.push('grep=' + grep);
        }
        if (tailLines >= 0) {
            params.push('podLogOptions.tailLines=' + tailLines);
        }
        return requests.loadEventSource(`api/v1/stream/pipelines/${namespace}/logs?${params.join('&')}`).map(line => line && (JSON.parse(line).result as LogEntry));
    }

    public getPipeline(namespace: string, name: string) {
        return requests.get(`api/v1/pipelines/${namespace}/${name}`).then(res => res.body as Pipeline);
    }

    public restartPipeline(namespace: string, name: string) {
        return requests.post(`api/v1/pipelines/${namespace}/${name}/restart`);
    }
    public deletePipeline(namespace: string, name: string) {
        return requests.delete(`api/v1/pipelines/${namespace}/${name}`);
    }

    public watchSteps(namespace: string, labels?: Array<string>) {
        return requests
            .loadEventSource(`api/v1/stream/steps/${namespace}?listOptions.labelSelector=${labels.join(',')}`)
            .map(line => line && (JSON.parse(line).result as StepWatchEvent));
    }
}

import {APIRequestContext, expect} from '@playwright/test';

import {bearer, ENV_FACTOR, NAMESPACE} from './auth';
import type {TestResource} from './workflows';

// The REST collection each kind of resource lives under, the request-body field
// the server's create endpoint expects it in, and whether the collection is
// namespaced.
const KINDS = {
    workflow: {collection: 'workflows', field: 'workflow', namespaced: true},
    workflowTemplate: {collection: 'workflow-templates', field: 'template', namespaced: true},
    clusterWorkflowTemplate: {collection: 'cluster-workflow-templates', field: 'template', namespaced: false},
    cronWorkflow: {collection: 'cron-workflows', field: 'cronWorkflow', namespaced: true}
} as const;
export type Kind = keyof typeof KINDS;

// Seeds and inspects backend state directly over the REST API (proxied through
// the webpack dev server at /api/v1). Tests prefer submitting here and waiting
// for a terminal phase before touching the browser, so rendering assertions do
// not race the controller.
export class ApiClient {
    private readonly headers: Record<string, string>;
    private readonly created: {kind: Kind; name: string}[] = [];

    constructor(
        private readonly request: APIRequestContext,
        readonly namespace = NAMESPACE
    ) {
        this.headers = {Authorization: bearer()};
    }

    /** The API path of `kind`'s collection, or of the resource `name` within it. */
    private path(kind: Kind, name?: string): string {
        const {collection, namespaced} = KINDS[kind];
        return ['api/v1', collection, ...(namespaced ? [this.namespace] : []), ...(name ? [name] : [])].join('/');
    }

    /** Creates `resource` as a `kind` and returns its server-generated name. */
    private async create(kind: Kind, resource: TestResource): Promise<string> {
        const res = await this.request.post(this.path(kind), {headers: this.headers, data: {[KINDS[kind].field]: resource}});
        expect(res.ok(), `create ${kind} failed: ${res.status()} ${await res.text()}`).toBeTruthy();
        const name = (await res.json()).metadata.name as string;
        this.track(name, kind);
        return name;
    }

    /** Submits a workflow and returns its server-generated name. */
    async submitWorkflow(workflow: TestResource): Promise<string> {
        return this.create('workflow', workflow);
    }

    /** Creates a WorkflowTemplate, or a ClusterWorkflowTemplate when `clusterScoped`. */
    async createWorkflowTemplate(template: TestResource, clusterScoped = false): Promise<string> {
        return this.create(clusterScoped ? 'clusterWorkflowTemplate' : 'workflowTemplate', template);
    }

    async createCronWorkflow(cronWorkflow: TestResource): Promise<string> {
        return this.create('cronWorkflow', cronWorkflow);
    }

    /**
     * Registers a resource this client did not create — one the browser created,
     * whose name the server or the UI generated — so teardown cleans it up too.
     */
    track(name: string, kind: Kind = 'workflow'): void {
        this.created.push({kind, name});
    }

    async getWorkflow(name: string): Promise<any> {
        const res = await this.request.get(this.path('workflow', name), {headers: this.headers});
        expect(res.ok(), `get ${name} failed: ${res.status()}`).toBeTruthy();
        return res.json();
    }

    /**
     * Polls until the workflow reaches one of `phases` (default 30s, scaled by
     * E2E_ENV_FACTOR). Fails fast — with the workflow's message — if it reaches
     * a different terminal phase, rather than burning the whole timeout and
     * reporting a misleading "never reached" error.
     */
    async waitForPhase(name: string, phases: string | string[], timeoutMs = 30_000): Promise<void> {
        const wanted = new Set(Array.isArray(phases) ? phases : [phases]);
        const terminal = new Set(['Succeeded', 'Failed', 'Error']);
        const deadline = Date.now() + timeoutMs * ENV_FACTOR;
        let phase = 'Pending';
        while (Date.now() < deadline) {
            const wf = await this.getWorkflow(name);
            phase = wf.status?.phase ?? 'Pending';
            if (wanted.has(phase)) {
                return;
            }
            if (terminal.has(phase)) {
                throw new Error(`workflow ${name} reached terminal phase ${phase}, wanted ${[...wanted].join('/')}: ${wf.status?.message ?? '(no message)'}`);
            }
            await new Promise(resolve => setTimeout(resolve, 1_000));
        }
        throw new Error(`workflow ${name} never reached ${[...wanted].join('/')} within ${timeoutMs * ENV_FACTOR}ms (last phase: ${phase})`);
    }

    async getCronWorkflow(name: string): Promise<any> {
        const res = await this.request.get(this.path('cronWorkflow', name), {headers: this.headers});
        expect(res.ok(), `get cron workflow ${name} failed: ${res.status()}`).toBeTruthy();
        return res.json();
    }

    private async delete(kind: Kind, name: string): Promise<void> {
        await this.request.delete(this.path(kind, name), {headers: this.headers});
    }

    /**
     * Removes everything this client created or tracked; called on fixture
     * teardown. Workflows a cron workflow spawned carry an owner reference to
     * it, so Kubernetes garbage-collects them along with it.
     */
    async cleanup(): Promise<void> {
        await Promise.all(this.created.splice(0).map(({kind, name}) => this.delete(kind, name).catch(() => undefined)));
    }
}

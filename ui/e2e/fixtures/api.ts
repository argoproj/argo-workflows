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

/** The label the controller sets once it has copied a completed workflow into the archive. */
const ARCHIVING_STATUS = 'workflows.argoproj.io/workflow-archiving-status';

// Whether the server has the workflow archive, probed once per worker.
let archiveEnabled: Promise<boolean> | undefined;

const TERMINAL_PHASES = new Set(['Succeeded', 'Failed', 'Error']);

// Seeds and inspects backend state directly over the REST API (proxied through
// the webpack dev server at /api/v1). Tests prefer submitting here and waiting
// for a terminal phase before touching the browser, so rendering assertions do
// not race the controller.
export class ApiClient {
    private readonly headers: Record<string, string>;
    private readonly created: {kind: Kind; name: string; uid?: string}[] = [];

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
        const {name, uid} = (await res.json()).metadata;
        this.created.push({kind, name, uid});
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
     *
     * With the archive enabled, reaching a terminal phase also waits for the
     * controller to archive the workflow. Until then the list can show it twice
     * (the controller writes the archive before relabelling the live
     * workflow), so a page rendered sooner would race the controller.
     */
    async waitForPhase(name: string, phases: string | string[], timeoutMs = 30_000): Promise<void> {
        const wanted = new Set(Array.isArray(phases) ? phases : [phases]);
        const deadline = Date.now() + timeoutMs * ENV_FACTOR;
        let phase = 'Pending';
        while (Date.now() < deadline) {
            const wf = await this.getWorkflow(name);
            phase = wf.status?.phase ?? 'Pending';
            if (wanted.has(phase)) {
                if (TERMINAL_PHASES.has(phase) && (await this.archiveEnabled())) {
                    await this.waitForArchived(name);
                }
                return;
            }
            if (TERMINAL_PHASES.has(phase)) {
                throw new Error(`workflow ${name} reached terminal phase ${phase}, wanted ${[...wanted].join('/')}: ${wf.status?.message ?? '(no message)'}`);
            }
            await new Promise(resolve => setTimeout(resolve, 1_000));
        }
        throw new Error(`workflow ${name} never reached ${[...wanted].join('/')} within ${timeoutMs * ENV_FACTOR}ms (last phase: ${phase})`);
    }

    /**
     * Whether the server has the workflow archive (`persistence.archive`, as the
     * mysql and postgres profiles enable). Asking for an archived workflow that
     * does not exist tells them apart: not found with the archive, "not
     * supported" without it.
     */
    async archiveEnabled(): Promise<boolean> {
        archiveEnabled ??= (async () => {
            const res = await this.request.get(`api/v1/archived-workflows/00000000-0000-0000-0000-000000000000?namespace=${this.namespace}`, {headers: this.headers});
            if (res.status() === 404) {
                return true;
            }
            const body = await res.text();
            if (body.includes('not supported')) {
                return false;
            }
            throw new Error(`cannot tell whether the workflow archive is enabled: ${res.status()} ${body}`);
        })();
        return archiveEnabled;
    }

    /**
     * Polls until the controller has copied the (completed) workflow into the
     * archive: `Archived` on the live workflow, or `Persisted` once only the
     * archived copy is left.
     */
    async waitForArchived(name: string, timeoutMs = 30_000): Promise<void> {
        await expect
            .poll(async () => (await this.getWorkflow(name)).metadata.labels?.[ARCHIVING_STATUS], {
                message: `workflow ${name} was never archived`,
                timeout: timeoutMs * ENV_FACTOR
            })
            .toMatch(/^(Archived|Persisted)$/);
    }

    /**
     * Deletes an archived workflow from the cluster, leaving its archived copy,
     * and waits until it is gone. Getting a workflow that is not in the cluster
     * falls back to the archive, so "gone" shows as the server answering with
     * the archived copy, which it labels `Persisted` (the live one says
     * `Archived`).
     */
    async deleteWorkflowFromCluster(name: string): Promise<void> {
        await this.delete('workflow', name);
        await expect
            .poll(async () => (await this.getWorkflow(name)).metadata.labels?.[ARCHIVING_STATUS], {
                message: `workflow ${name} was never deleted from the cluster`,
                timeout: 30_000 * ENV_FACTOR
            })
            .toBe('Persisted');
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
     * Deletes a workflow and, with the archive enabled, its archived copy too,
     * so archived rows do not pile up in the list across tests and passes. A
     * completed workflow may not be archived yet (a test can end the moment the
     * page shows the final phase), and archiving it after its copy was deleted
     * would leave the copy behind, so wait for that first.
     */
    private async deleteWorkflowEverywhere(name: string, uid?: string): Promise<void> {
        if (!(await this.archiveEnabled())) {
            return this.delete('workflow', name);
        }
        const res = await this.request.get(this.path('workflow', name), {headers: this.headers});
        if (res.ok()) {
            const wf = await res.json();
            uid ??= wf.metadata.uid;
            if (TERMINAL_PHASES.has(wf.status?.phase)) {
                await this.waitForArchived(name, 10_000).catch(() => undefined);
            }
        }
        await this.delete('workflow', name);
        if (uid) {
            await this.request.delete(`api/v1/archived-workflows/${uid}?namespace=${this.namespace}`, {headers: this.headers});
        }
    }

    /**
     * Removes everything this client created or tracked; called on fixture
     * teardown. Workflows a cron workflow spawned carry an owner reference to
     * it, so Kubernetes garbage-collects them along with it.
     */
    async cleanup(): Promise<void> {
        await Promise.all(
            this.created.splice(0).map(({kind, name, uid}) => (kind === 'workflow' ? this.deleteWorkflowEverywhere(name, uid) : this.delete(kind, name)).catch(() => undefined))
        );
    }
}

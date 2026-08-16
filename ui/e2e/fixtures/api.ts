import {APIRequestContext, expect} from '@playwright/test';

import {bearer, ENV_FACTOR, NAMESPACE} from './auth';
import type {TestWorkflow} from './workflows';

// Seeds and inspects backend state directly over the REST API (proxied through
// the webpack dev server at /api/v1). Tests prefer submitting here and waiting
// for a terminal phase before touching the browser, so rendering assertions do
// not race the controller.
export class ApiClient {
    private readonly headers: Record<string, string>;
    private readonly created: string[] = [];

    constructor(
        private readonly request: APIRequestContext,
        readonly namespace = NAMESPACE
    ) {
        this.headers = {Authorization: bearer()};
    }

    /** Submits a workflow and returns its server-generated name. */
    async submitWorkflow(workflow: TestWorkflow): Promise<string> {
        const res = await this.request.post(`api/v1/workflows/${this.namespace}`, {headers: this.headers, data: {workflow}});
        expect(res.ok(), `submit failed: ${res.status()} ${await res.text()}`).toBeTruthy();
        const name = (await res.json()).metadata.name as string;
        this.created.push(name);
        return name;
    }

    async getWorkflow(name: string): Promise<any> {
        const res = await this.request.get(`api/v1/workflows/${this.namespace}/${name}`, {headers: this.headers});
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

    async deleteWorkflow(name: string): Promise<void> {
        await this.request.delete(`api/v1/workflows/${this.namespace}/${name}`, {headers: this.headers});
    }

    /** Removes every workflow this client created; called on fixture teardown. */
    async cleanup(): Promise<void> {
        await Promise.all(this.created.splice(0).map(name => this.deleteWorkflow(name).catch(() => undefined)));
    }
}

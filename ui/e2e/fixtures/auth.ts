import {execFileSync} from 'child_process';

// The UI is exercised in `client` auth mode: the argo-server reads an
// `authorization` cookie whose value is a `Bearer <token>` string. We source
// that token from the same place the Go e2e suite does — the
// `argo-server.service-account-token` secret in the argo namespace
// (see test/e2e/fixtures/e2e_suite.go). Set ARGO_TOKEN to override, e.g. when
// running against a cluster you cannot `kubectl get secret` on.

export const NAMESPACE = process.env.ARGO_NAMESPACE || 'argo';
export const BASE_URL = process.env.ARGO_UI_BASE_URL || 'http://localhost:8080';
export const ENV_FACTOR = Number(process.env.E2E_ENV_FACTOR || '1');

const SECRET_NAME = 'argo-server.service-account-token';

let cached: string | undefined;

/** The raw bearer token (no `Bearer ` prefix). */
export function resolveToken(): string {
    if (cached) {
        return cached;
    }
    if (process.env.ARGO_TOKEN) {
        cached = process.env.ARGO_TOKEN.replace(/^Bearer\s+/i, '').trim();
        return cached;
    }
    const b64 = execFileSync('kubectl', ['-n', NAMESPACE, 'get', 'secret', SECRET_NAME, '-o', 'jsonpath={.data.token}'], {encoding: 'utf-8'}).trim();
    if (!b64) {
        throw new Error(`empty token in secret ${SECRET_NAME} (namespace ${NAMESPACE}); is the argo-server service account created? Set ARGO_TOKEN to override.`);
    }
    cached = Buffer.from(b64, 'base64').toString('utf-8').trim();
    return cached;
}

/** The value to store in the `authorization` cookie / send as the Authorization header. */
export function bearer(): string {
    return `Bearer ${resolveToken()}`;
}

import requests from '../shared/services/requests';

export async function request(namespace: string, path: string, method = 'get', body?: any) {
    const url = 'api/v1/parameterization/' + encodeURIComponent(namespace) + '/' + path;
    const response = method === 'get' ? await requests.get(url) : method === 'put' ? await requests.put(url).send(body) : await requests.post(url).send(body);
    return response.body;
}

export interface CapabilityReport {
    experimental: boolean;
    schemaDialect: string;
    nativeWorkflowOwner: 'Argo';
    features: string[];
    unsupported: string[];
    encryptedSecretInput: boolean;
    actionCallScopes: string[];
    secretVersionPolicy: string;
    partialRetryScopes: string[];
}

export interface ActionExecutionContract {
    effect: 'Pure' | 'ReadOnly' | 'Mutating';
    targetScope: string;
    permissionScope: string;
    outputMeaning: string;
    capabilities?: Array<{name: 'Idempotency' | 'Observe' | 'Cancel' | 'Recover' | 'ResultReuse'; condition: string}>;
}

export interface ActionReview {
    implementationDigest: string;
    definition: {
        executionContract?: ActionExecutionContract;
        inputSchema: object;
        outputSchema: object;
        template: object;
    };
    versions: {[name: string]: string};
    verification: string;
}

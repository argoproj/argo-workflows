// Minimal, deterministic workflow manifests for tests. Kept as typed builders
// (rather than YAML files) so they are dependency-free and refactor-safe.
// All use argosay, the lightweight image the Go e2e suite relies on.

import {NAMESPACE} from './auth';

export const TEST_LABEL = 'workflows.argoproj.io/test';

export interface TestWorkflow {
    metadata: {generateName?: string; name?: string; namespace?: string; labels?: Record<string, string>};
    spec: unknown;
}

function base(generateName: string): Pick<TestWorkflow, 'metadata'> {
    return {metadata: {generateName, namespace: NAMESPACE, labels: {[TEST_LABEL]: 'true'}}};
}

/** A workflow that echoes `message` and succeeds within a few seconds. */
export function echoWorkflow(message = 'hello e2e', generateName = 'e2e-smoke-'): TestWorkflow {
    return {
        ...base(generateName),
        spec: {
            entrypoint: 'main',
            templates: [{name: 'main', container: {image: 'argoproj/argosay:v2', args: ['echo', message]}}]
        }
    };
}

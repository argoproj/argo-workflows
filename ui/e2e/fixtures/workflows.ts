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

const ARGOSAY = 'argoproj/argosay:v2';

/** A single-container workflow running `argosay <args>`. */
function oneStep(generateName: string, args: string[]): TestWorkflow {
    return {
        ...base(generateName),
        spec: {
            entrypoint: 'main',
            templates: [{name: 'main', container: {image: ARGOSAY, args}}]
        }
    };
}

/** A workflow that echoes `message` and succeeds within a few seconds. */
export function echoWorkflow(message = 'hello e2e', generateName = 'e2e-smoke-'): TestWorkflow {
    return oneStep(generateName, ['echo', message]);
}

/** A workflow whose only step exits non-zero, so it settles in `Failed`. */
export function failingWorkflow(generateName = 'e2e-fail-'): TestWorkflow {
    return oneStep(generateName, ['exit', '1']);
}

/** A workflow that stays `Running` for `seconds`, for tests that act on a live workflow. */
export function sleepWorkflow(seconds = 60, generateName = 'e2e-sleep-'): TestWorkflow {
    return oneStep(generateName, ['sleep', String(seconds)]);
}

/** The task names in `dagWorkflow`, which double as its graph node labels. */
export const DAG_TASKS = ['generate', 'process-a', 'process-b'];

/** A DAG that fans `generate` out into two parallel tasks and succeeds. */
export function dagWorkflow(generateName = 'e2e-dag-'): TestWorkflow {
    const [generate, ...processors] = DAG_TASKS;
    return {
        ...base(generateName),
        spec: {
            entrypoint: 'main',
            templates: [
                {
                    name: 'main',
                    dag: {
                        tasks: [{name: generate, template: 'echo'}, ...processors.map(name => ({name, template: 'echo', dependencies: [generate]}))]
                    }
                },
                {name: 'echo', container: {image: ARGOSAY, args: ['echo', 'hello e2e']}}
            ]
        }
    };
}

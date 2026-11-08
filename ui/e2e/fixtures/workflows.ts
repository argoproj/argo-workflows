// Minimal, deterministic workflow (and workflow template and cron workflow)
// manifests for tests. Kept as typed builders
// (rather than YAML files) so they are dependency-free and refactor-safe.
// All use argosay, the lightweight image the Go e2e suite relies on.

import {NAMESPACE} from './auth';

export const TEST_LABEL = 'workflows.argoproj.io/test';

export interface TestResource {
    metadata: {generateName?: string; name?: string; namespace?: string; labels?: Record<string, string>};
    spec: unknown;
}

function base(generateName: string): Pick<TestResource, 'metadata'> {
    return {metadata: {generateName, namespace: NAMESPACE, labels: {[TEST_LABEL]: 'true'}}};
}

const ARGOSAY = 'argoproj/argosay:v2';

/** The spec of a single-container workflow running `argosay <args>`. */
function oneStepSpec(args: string[]) {
    return {
        entrypoint: 'main',
        templates: [{name: 'main', container: {image: ARGOSAY, args}}]
    };
}

/** A single-container workflow running `argosay <args>`. */
function oneStep(generateName: string, args: string[]): TestResource {
    return {...base(generateName), spec: oneStepSpec(args)};
}

/** A workflow that echoes `message` and succeeds within a few seconds. */
export function echoWorkflow(message = 'hello e2e', generateName = 'e2e-smoke-'): TestResource {
    return oneStep(generateName, ['echo', message]);
}

/** A workflow whose only step exits non-zero, so it settles in `Failed`. */
export function failingWorkflow(generateName = 'e2e-fail-'): TestResource {
    return oneStep(generateName, ['exit', '1']);
}

/** A workflow that stays `Running` for `seconds`, for tests that act on a live workflow. */
export function sleepWorkflow(seconds = 60, generateName = 'e2e-sleep-'): TestResource {
    return oneStep(generateName, ['sleep', String(seconds)]);
}

/** The task names in `dagWorkflow`, which double as its graph node labels. */
export const DAG_TASKS = ['generate', 'process-a', 'process-b'];

/** A DAG that fans `generate` out into two parallel tasks and succeeds. */
export function dagWorkflow(generateName = 'e2e-dag-'): TestResource {
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

/** The `message` parameter's default in `echoWorkflowTemplate`. */
export const TEMPLATE_MESSAGE = 'hello e2e template';

/**
 * A WorkflowTemplate whose workflows echo their `message` parameter and succeed,
 * or, when `clusterScoped`, the same as a ClusterWorkflowTemplate.
 */
export function echoWorkflowTemplate(clusterScoped = false, generateName = 'e2e-template-'): TestResource {
    const {metadata} = base(generateName);
    if (clusterScoped) {
        delete metadata.namespace;
    }
    return {
        metadata,
        spec: {
            ...oneStepSpec(['echo', '{{workflow.parameters.message}}']),
            arguments: {parameters: [{name: 'message', value: TEMPLATE_MESSAGE}]}
        }
    };
}

/**
 * A cron workflow running `echoWorkflow`'s step. The default schedule (midnight
 * on 1 January) is valid but never due mid-test, so it spawns no workflows.
 */
export function cronWorkflow(schedule = '0 0 1 1 *', generateName = 'e2e-cron-'): TestResource {
    return {...base(generateName), spec: {schedules: [schedule], workflowSpec: oneStepSpec(['echo', 'hello e2e'])}};
}

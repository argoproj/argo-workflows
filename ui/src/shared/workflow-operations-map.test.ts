import {Workflow} from './models';
import {WorkflowOperationsMap} from './workflow-operations-map';

const suspendedWorkflow = (shutdown?: 'Terminate' | 'Stop'): Workflow =>
    ({
        metadata: {name: 'hello-world', namespace: 'argo'},
        spec: {suspend: true, shutdown},
        status: {phase: 'Running'}
    }) as Workflow;

const terminatedWorkflow = (shutdown?: 'Terminate' | 'Stop'): Workflow =>
    ({
        metadata: {name: 'hello-world', namespace: 'argo'},
        spec: {suspend: true, shutdown},
        status: {phase: 'Failed'}
    }) as Workflow;

describe('WorkflowOperationsMap RESUME', () => {
    test('enabled for a suspended workflow with no shutdown strategy', () => {
        expect(WorkflowOperationsMap.RESUME.disabled(suspendedWorkflow())).toBe(false);
    });

    test('enabled when the workflow is being terminated', () => {
        expect(WorkflowOperationsMap.RESUME.disabled(suspendedWorkflow('Terminate'))).toBe(false);
    });

    test('enabled once the workflow is being stopped', () => {
        expect(WorkflowOperationsMap.RESUME.disabled(suspendedWorkflow('Stop'))).toBe(false);
    });

    test('disabled when the workflow is not suspended', () => {
        const wf = suspendedWorkflow();
        wf.spec.suspend = false;
        expect(WorkflowOperationsMap.RESUME.disabled(wf)).toBe(true);
    });

    test('disabled once the workflow is terminated', () => {
        expect(WorkflowOperationsMap.RESUME.disabled(terminatedWorkflow('Terminate'))).toBe(true);
    });

    test('disabled once the workflow is stopped', () => {
        expect(WorkflowOperationsMap.RESUME.disabled(terminatedWorkflow('Stop'))).toBe(true);
    });
});

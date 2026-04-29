import {render, waitFor} from '@testing-library/react';
import React from 'react';

import {exampleClusterWorkflowTemplate, exampleCronWorkflow, exampleWorkflow, exampleWorkflowTemplate} from '../../examples';
import {Workflow} from '../../models';
import {services} from '../../services';
import {
    convertFromCronWorkflow,
    generateNamePostfix,
    getLeafNodes,
    getStringAfterDelimiter,
    GraphViewer,
    parseDepends,
    populateGraphFromWorkflow
} from './graph-viewer';

jest.mock('../../services/workflow-template-service');
jest.mock('../../services/cluster-workflow-template-service');

// GraphPanel renders SVG/canvas which is not available in jsdom; mock it to keep tests focused on GraphViewer logic.
jest.mock('../graph/graph-panel', () => ({
    GraphPanel: (props: {graph: {nodes: Map<string, unknown>; edges: Map<unknown, unknown>}}) => (
        <div data-testid='graph-panel' data-node-count={props.graph.nodes.size} data-edge-count={props.graph.edges.size} />
    )
}));

describe('parseDepends', () => {
    it('extracts simple task names', () => {
        expect(parseDepends('taskA && taskB')).toEqual(expect.arrayContaining(['taskA', 'taskB']));
    });

    it('handles task.Success and task.Failed suffixes', () => {
        const result = parseDepends('taskA.Succeeded || taskB.Failed');
        expect(result).toContain('taskA');
        expect(result).toContain('taskB');
        // suffixes should not appear as separate entries
        expect(result).not.toContain('Succeeded');
        expect(result).not.toContain('Failed');
    });

    it('deduplicates repeated task names', () => {
        const result = parseDepends('taskA && taskA');
        expect(result.filter(t => t === 'taskA')).toHaveLength(1);
    });

    it('returns empty array for empty string', () => {
        expect(parseDepends('')).toEqual([]);
    });
});

describe('generateNamePostfix', () => {
    it('generates a string of the requested length', () => {
        expect(generateNamePostfix(5)).toHaveLength(5);
        expect(generateNamePostfix(10)).toHaveLength(10);
    });

    it('only contains lowercase letters and digits', () => {
        const result = generateNamePostfix(50);
        expect(result).toMatch(/^[a-z0-9]+$/);
    });

    it('generates different values on successive calls (probabilistic)', () => {
        const a = generateNamePostfix(10);
        const b = generateNamePostfix(10);
        // Probability of collision is ~36^-10 ≈ 0
        expect(a).not.toBe(b);
    });
});

describe('getStringAfterDelimiter', () => {
    it('returns the part after the last dot', () => {
        expect(getStringAfterDelimiter('a.b.c')).toBe('c');
    });

    it('returns the whole string when there is no delimiter', () => {
        expect(getStringAfterDelimiter('abc')).toBe('abc');
    });

    it('supports custom delimiter', () => {
        expect(getStringAfterDelimiter('a/b/c', '/')).toBe('c');
    });
});

describe('getLeafNodes', () => {
    it('returns all DAG tasks that are not depended upon', () => {
        const template = {
            name: 'dag-tmpl',
            dag: {
                tasks: [
                    {name: 'step-a', template: 'pod-tmpl'},
                    {name: 'step-b', template: 'pod-tmpl', depends: 'step-a'}
                ]
            }
        };
        const leaves = getLeafNodes(template as any);
        expect(leaves).toContain('step-b');
        expect(leaves).not.toContain('step-a');
    });

    it('returns the last step group entries for a steps template', () => {
        const template = {
            name: 'steps-tmpl',
            steps: [
                [{name: 'step-0', template: 'pod-tmpl'}],
                [{name: 'step-1a', template: 'pod-tmpl'}, {name: 'step-1b', template: 'pod-tmpl'}]
            ]
        };
        const leaves = getLeafNodes(template as any);
        expect(leaves).toContain('1.step-1a');
        expect(leaves).toContain('1.step-1b');
        expect(leaves).not.toContain('0.step-0');
    });

    it('returns null for a plain pod template', () => {
        const template = {name: 'pod-tmpl', container: {image: 'alpine'}};
        expect(getLeafNodes(template as any)).toBeNull();
    });
});

describe('convertFromCronWorkflow', () => {
    it('converts a CronWorkflow into a Workflow with the inner workflowSpec', () => {
        const cron = exampleCronWorkflow('argo');
        const result = convertFromCronWorkflow(cron);
        expect(result.kind).toBe('Workflow');
        expect(result.spec).toBe(cron.spec.workflowSpec);
        expect(result.metadata).toBe(cron.metadata);
    });
});

describe('populateGraphFromWorkflow', () => {
    it('creates a Pod node for a single-template workflow', () => {
        const workflow = exampleWorkflow('argo');
        const graph = populateGraphFromWorkflow(workflow, 'test-run');
        expect(graph.nodes.size).toBeGreaterThan(0);
        // The single entrypoint template should be represented as a Pod node
        const genres = Array.from(graph.nodes.values()).map((n: any) => n.genre);
        expect(genres).toContain('Pod');
    });

    it('creates DAG nodes for a DAG workflow', () => {
        const workflow: Workflow = {
            metadata: {name: 'dag-wf', namespace: 'argo'},
            spec: {
                entrypoint: 'dag-tmpl',
                templates: [
                    {
                        name: 'dag-tmpl',
                        dag: {
                            tasks: [
                                {name: 'step-a', template: 'pod-tmpl'},
                                {name: 'step-b', template: 'pod-tmpl', depends: 'step-a'}
                            ]
                        }
                    },
                    {
                        name: 'pod-tmpl',
                        container: {name: 'main', image: 'alpine'}
                    }
                ]
            }
        };
        const graph = populateGraphFromWorkflow(workflow, 'dag-wf-run');
        const genres = Array.from(graph.nodes.values()).map((n: any) => n.genre);
        expect(genres).toContain('DAG');
        expect(genres).toContain('Pod');
    });

    it('creates Steps and StepGroup nodes for a steps workflow', () => {
        const workflow: Workflow = {
            metadata: {name: 'steps-wf', namespace: 'argo'},
            spec: {
                entrypoint: 'steps-tmpl',
                templates: [
                    {
                        name: 'steps-tmpl',
                        steps: [
                            [{name: 'step-a', template: 'pod-tmpl'}],
                            [{name: 'step-b', template: 'pod-tmpl'}]
                        ]
                    },
                    {
                        name: 'pod-tmpl',
                        container: {name: 'main', image: 'alpine'}
                    }
                ]
            }
        };
        const graph = populateGraphFromWorkflow(workflow, 'steps-wf-run');
        const genres = Array.from(graph.nodes.values()).map((n: any) => n.genre);
        expect(genres).toContain('Steps');
        expect(genres).toContain('StepGroup');
        expect(genres).toContain('Pod');
    });
});

describe('GraphViewer component', () => {
    beforeEach(() => {
        jest.clearAllMocks();
    });

    it('shows loading state initially for a workflow with a workflowTemplateRef', async () => {
        const workflow: Workflow = {
            metadata: {name: 'ref-wf', namespace: 'argo'},
            spec: {
                entrypoint: '',
                templates: [],
                workflowTemplateRef: {name: 'my-template'}
            } as any
        };

        jest.spyOn(services.workflowTemplate, 'get').mockReturnValue(new Promise(() => {})); // never resolves

        const {getByText} = render(<GraphViewer workflowDefinition={workflow} />);
        expect(getByText('Loading...')).toBeInTheDocument();
    });

    it('renders GraphPanel after loading a plain workflow', async () => {
        const workflow = exampleWorkflow('argo');
        const {getByTestId} = render(<GraphViewer workflowDefinition={workflow} />);
        await waitFor(() => {
            expect(getByTestId('graph-panel')).toBeInTheDocument();
        });
    });

    it('renders GraphPanel for a WorkflowTemplate', async () => {
        const wft = exampleWorkflowTemplate('argo');
        const {getByTestId} = render(<GraphViewer workflowDefinition={wft} />);
        await waitFor(() => {
            expect(getByTestId('graph-panel')).toBeInTheDocument();
        });
    });

    it('renders GraphPanel for a ClusterWorkflowTemplate', async () => {
        const cwft = exampleClusterWorkflowTemplate();
        const {getByTestId} = render(<GraphViewer workflowDefinition={cwft} />);
        await waitFor(() => {
            expect(getByTestId('graph-panel')).toBeInTheDocument();
        });
    });

    it('converts and renders a CronWorkflow', async () => {
        const cron = exampleCronWorkflow('argo');
        const {getByTestId} = render(<GraphViewer workflowDefinition={cron} />);
        await waitFor(() => {
            expect(getByTestId('graph-panel')).toBeInTheDocument();
        });
    });

    it('resolves workflowTemplateRef via service and then renders graph', async () => {
        const wft = exampleWorkflowTemplate('argo');
        jest.spyOn(services.workflowTemplate, 'get').mockResolvedValue(wft);

        const workflow: Workflow = {
            metadata: {name: 'ref-wf', namespace: 'argo'},
            spec: {
                entrypoint: '',
                templates: [],
                workflowTemplateRef: {name: wft.metadata.name}
            } as any
        };

        const {getByTestId} = render(<GraphViewer workflowDefinition={workflow} />);
        await waitFor(() => {
            expect(getByTestId('graph-panel')).toBeInTheDocument();
        });
        expect(services.workflowTemplate.get).toHaveBeenCalledWith(wft.metadata.name, 'argo');
    });

    it('resolves clusterScope workflowTemplateRef via clusterWorkflowTemplate service', async () => {
        const cwft = exampleClusterWorkflowTemplate();
        jest.spyOn(services.clusterWorkflowTemplate, 'get').mockResolvedValue(cwft);

        const workflow: Workflow = {
            metadata: {name: 'cluster-ref-wf', namespace: 'argo'},
            spec: {
                entrypoint: '',
                templates: [],
                workflowTemplateRef: {name: cwft.metadata.name, clusterScope: true}
            } as any
        };

        const {getByTestId} = render(<GraphViewer workflowDefinition={workflow} />);
        await waitFor(() => {
            expect(getByTestId('graph-panel')).toBeInTheDocument();
        });
        expect(services.clusterWorkflowTemplate.get).toHaveBeenCalledWith(cwft.metadata.name);
    });

    it('shows an error message when workflowTemplateRef resolution fails', async () => {
        jest.spyOn(services.workflowTemplate, 'get').mockRejectedValue(new Error('not found'));

        const workflow: Workflow = {
            metadata: {name: 'bad-ref-wf', namespace: 'argo'},
            spec: {
                entrypoint: '',
                templates: [],
                workflowTemplateRef: {name: 'missing-template'}
            } as any
        };

        const {getByText} = render(<GraphViewer workflowDefinition={workflow} />);
        await waitFor(() => {
            expect(getByText(/not found/)).toBeInTheDocument();
        });
    });

    it('uses generateName when metadata.name is absent', async () => {
        const workflow: Workflow = {
            metadata: {generateName: 'my-prefix-', namespace: 'argo'},
            spec: {
                entrypoint: 'argosay',
                templates: [
                    {
                        name: 'argosay',
                        container: {name: 'main', image: 'argoproj/argosay:v2'}
                    }
                ]
            }
        };

        const {getByTestId} = render(<GraphViewer workflowDefinition={workflow} />);
        await waitFor(() => {
            expect(getByTestId('graph-panel')).toBeInTheDocument();
        });
    });
});

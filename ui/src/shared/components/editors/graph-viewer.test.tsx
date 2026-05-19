import {render} from '@testing-library/react';
import React from 'react';

import {GraphViewer} from './graph-viewer';

jest.mock('../graph/graph-panel', () => ({
    GraphPanel: ({graph}: {graph?: any}) => <div data-testid='graph'>{graph ? 'rendered' : 'missing'}</div>
}));

describe('GraphViewer', () => {
    it('does not crash when DAG dependency points to non-existing node', async () => {
        const workflow = {
            metadata: {
                name: 'cronwf'
            },
            spec: {
                entrypoint: 'main',
                templates: [
                    {
                        name: 'main',
                        dag: {
                            tasks: [
                                {
                                    name: 'build',
                                    template: 'build-template'
                                },
                                {
                                    name: 'transform',
                                    template: 'transform-template',
                                    depends: 'build && analyze'
                                }
                            ]
                        }
                    },
                    {
                        name: 'build-template',
                        container: {}
                    },
                    {
                        name: 'transform-template',
                        container: {}
                    }
                ]
            }
        };

        const {findByTestId} = render(<GraphViewer workflowDefinition={workflow as any} />);

        const graph = await findByTestId('graph');
        expect(graph.textContent).toBe('rendered');
    });
});

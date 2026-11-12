import {render, screen} from '@testing-library/react';
import {createMemoryHistory} from 'history';
import React from 'react';

import {Context} from '../../shared/context';
import {Artifact} from '../../shared/models';
import {SubmitWorkflowPanel} from './submit-workflow-panel';

jest.mock('../../shared/components/tooltip', () => ({
    Tooltip: ({content, children}: {content: string; children: React.ReactNode}) => (
        <span data-testid='artifact-description-tooltip' data-content={content}>
            {children}
        </span>
    )
}));

function renderPanel(workflowArtifacts: Artifact[]) {
    const history = createMemoryHistory();
    return render(
        <Context.Provider value={{navigation: {goto: jest.fn()}} as any}>
            <SubmitWorkflowPanel
                kind='WorkflowTemplate'
                namespace='argo'
                name='artifact-template'
                entrypoint=''
                templates={[]}
                workflowParameters={[]}
                workflowArtifacts={workflowArtifacts}
                history={history}
            />
        </Context.Provider>
    );
}

describe('SubmitWorkflowPanel artifact descriptions', () => {
    it('shows a tooltip only for artifacts with descriptions', () => {
        const {container} = renderPanel([
            {name: 'documented-artifact', description: 'Upload a ZIP archive'},
            {name: 'undocumented-artifact'},
            {name: 'empty-description-artifact', description: ''}
        ]);

        expect(screen.getByText('documented-artifact')).toBeInTheDocument();
        expect(screen.getByText('undocumented-artifact')).toBeInTheDocument();
        expect(screen.getByText('empty-description-artifact')).toBeInTheDocument();

        const tooltips = screen.getAllByTestId('artifact-description-tooltip');
        expect(tooltips).toHaveLength(1);
        expect(tooltips[0]).toHaveAttribute('data-content', 'Upload a ZIP archive');
        expect(container.querySelectorAll('.fa-question-circle')).toHaveLength(1);
    });
});

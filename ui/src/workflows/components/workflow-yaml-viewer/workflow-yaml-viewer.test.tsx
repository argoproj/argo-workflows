import {render} from '@testing-library/react';
import React from 'react';

import * as models from '../../../shared/models';
import {WorkflowYamlViewer} from './workflow-yaml-viewer';

jest.mock('argo-ui/src/components/slide-contents/slide-contents', () => ({
    SlideContents: (props: {title: string; contents: React.ReactNode}) => <div data-testid={`section-${props.title.replace(/\s+/g, '-')}`}>{props.contents}</div>
}));

jest.mock('../../../shared/components/object-editor', () => ({
    SerializingObjectEditor: (props: {value: unknown}) => <div data-testid='object-editor'>{JSON.stringify(props.value)}</div>
}));

function makeWorkflow(spec: {templates?: unknown[]}, status: {storedTemplateSpecs?: {uid: string}; storedTemplates?: unknown}): models.Workflow {
    return {spec, status} as unknown as models.Workflow;
}

function renderViewer(workflow: models.Workflow) {
    return render(<WorkflowYamlViewer workflow={workflow} selectedNode={null as unknown as models.NodeStatus} />);
}

describe('WorkflowYamlViewer stored templates', () => {
    const storedTemplates = {'namespaced/my-wft/step-template': {name: 'step-template'}};

    it('shows stored templates for a templateRef workflow with no offload marker', () => {
        const {queryByTestId} = renderViewer(makeWorkflow({templates: []}, {storedTemplates}));
        expect(queryByTestId('section-Stored-Templates')).not.toBeNull();
    });

    it('hides stored templates when the spec already holds the hydrated copy', () => {
        const {queryByTestId} = renderViewer(makeWorkflow({templates: [{name: 'main'}]}, {storedTemplateSpecs: {uid: 'uid-1'}, storedTemplates}));
        expect(queryByTestId('section-Stored-Templates')).toBeNull();
    });

    it('shows stored templates for an offloaded workflow before hydration fills the spec', () => {
        const {queryByTestId} = renderViewer(makeWorkflow({templates: []}, {storedTemplateSpecs: {uid: 'uid-1'}, storedTemplates}));
        expect(queryByTestId('section-Stored-Templates')).not.toBeNull();
    });

    it('hides the section when there are no stored templates', () => {
        const {queryByTestId} = renderViewer(makeWorkflow({templates: []}, {}));
        expect(queryByTestId('section-Stored-Templates')).toBeNull();
    });
});

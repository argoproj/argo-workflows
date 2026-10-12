import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import React from 'react';

import {request} from './api';
import {DefinitionEditor} from './definition-editor';

jest.mock('./api', () => ({request: jest.fn()}));
const doc = {kind: 'WorkflowForm', metadata: {name: 'form', uid: 'original', resourceVersion: '42'}, spec: {parameterTemplate: 'model'}};
beforeEach(() => (request as jest.Mock).mockReset());
it('validates without committing, then saves with the original UID/version and no metadata or status', async () => {
    (request as jest.Mock).mockResolvedValue({...doc, metadata: {...doc.metadata, resourceVersion: '43'}});
    const saved = jest.fn();
    render(<DefinitionEditor namespace='lab' resource='workflowforms' document={doc} onSaved={saved} />);
    fireEvent.click(screen.getByText('Edit definition spec'));
    expect(screen.getByLabelText('Definition name')).toBeDisabled();
    fireEvent.change(screen.getByLabelText('Definition spec YAML'), {target: {value: 'parameterTemplate: next'}});
    fireEvent.click(screen.getByText('Validate definition'));
    expect(await screen.findByRole('status')).toHaveTextContent('Installed CRD schema validation passed');
    expect(saved).not.toHaveBeenCalled();
    expect(request).toHaveBeenLastCalledWith('lab', 'definitions/workflowforms?name=form', 'put', {
        uid: 'original',
        resourceVersion: '42',
        validateOnly: true,
        spec: {parameterTemplate: 'next'}
    });
    fireEvent.click(screen.getByText('Save definition'));
    await waitFor(() => expect(saved).toHaveBeenCalledTimes(1));
    expect(request).toHaveBeenLastCalledWith('lab', 'definitions/workflowforms?name=form', 'put', {
        uid: 'original',
        resourceVersion: '42',
        validateOnly: false,
        spec: {parameterTemplate: 'next'}
    });
});
it('keeps conflicting edits for review and rejects invalid YAML without a request', async () => {
    (request as jest.Mock).mockRejectedValue({response: {body: {message: 'definition changed; reload'}}});
    render(<DefinitionEditor namespace='lab' resource='workflowforms' document={doc} onSaved={jest.fn()} />);
    fireEvent.click(screen.getByText('Edit definition spec'));
    fireEvent.change(screen.getByLabelText('Definition spec YAML'), {target: {value: 'layout: ['}});
    fireEvent.click(screen.getByText('Save definition'));
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(request).not.toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText('Definition spec YAML'), {target: {value: 'parameterTemplate: newer'}});
    fireEvent.click(screen.getByText('Save definition'));
    expect(await screen.findByRole('alert')).toHaveTextContent('definition changed');
    expect(screen.getByLabelText('Definition spec YAML')).toHaveValue('parameterTemplate: newer');
});
it('creates a new definition with empty preconditions and never exposes receipt editing', async () => {
    (request as jest.Mock).mockResolvedValue(doc);
    const saved = jest.fn();
    const view = render(<DefinitionEditor namespace='lab' resource='workflowforms' document={{metadata: {}, spec: {}}} onSaved={saved} />);
    fireEvent.change(screen.getByLabelText('Definition name'), {target: {value: 'new-form'}});
    fireEvent.click(screen.getByText('Save definition'));
    await waitFor(() => expect(saved).toHaveBeenCalled());
    expect(request).toHaveBeenLastCalledWith('lab', 'definitions/workflowforms?name=new-form', 'put', {spec: {}, uid: '', resourceVersion: '', validateOnly: false});
    view.rerender(<DefinitionEditor namespace='lab' resource='workflowsubmissions' document={doc} onSaved={saved} />);
    expect(screen.queryByRole('button')).toBeNull();
});
it('does not apply a save response after leaving the editor', async () => {
    let finish: (v: any) => void;
    (request as jest.Mock).mockReturnValue(
        new Promise(resolve => {
            finish = resolve;
        })
    );
    const saved = jest.fn();
    const view = render(<DefinitionEditor namespace='lab' resource='workflowforms' document={doc} onSaved={saved} />);
    fireEvent.click(screen.getByText('Edit definition spec'));
    fireEvent.click(screen.getByText('Save definition'));
    view.unmount();
    finish(doc);
    await waitFor(() => expect(saved).not.toHaveBeenCalled());
});
it('clones an immutable action into a new name instead of overwriting the published version', async () => {
    (request as jest.Mock).mockResolvedValue({...doc, metadata: {...doc.metadata, name: 'action-v2'}});
    const saved = jest.fn();
    render(<DefinitionEditor namespace='lab' resource='actiondefinitions' document={doc} onSaved={saved} />);
    expect(screen.queryByText('Edit definition spec')).toBeNull();
    fireEvent.click(screen.getByText('Clone as new definition'));
    fireEvent.change(screen.getByLabelText('Definition name'), {target: {value: 'action-v2'}});
    fireEvent.click(screen.getByText('Save definition'));
    await waitFor(() => expect(saved).toHaveBeenCalled());
    expect(request).toHaveBeenLastCalledWith('lab', 'definitions/actiondefinitions?name=action-v2', 'put', {spec: doc.spec, uid: '', resourceVersion: '', validateOnly: false});
});

it('restores an immutable model into a new editable name without copying its publication marker', async () => {
    (request as jest.Mock).mockResolvedValue({...doc, metadata: {...doc.metadata, name: 'restored'}});
    const saved = jest.fn();
    render(
        <DefinitionEditor
            namespace='lab'
            resource='parametertemplates'
            document={{...doc, metadata: {...doc.metadata, annotations: {'parameters.workflows.example.io/definition-digest': 'sealed'}}}}
            onSaved={saved}
        />
    );
    expect(screen.queryByText('Edit definition spec')).toBeNull();
    fireEvent.click(screen.getByText('Clone as new definition'));
    fireEvent.change(screen.getByLabelText('Definition name'), {target: {value: 'restored'}});
    fireEvent.click(screen.getByText('Save definition'));
    await waitFor(() => expect(saved).toHaveBeenCalled());
    expect(request).toHaveBeenLastCalledWith('lab', 'definitions/parametertemplates?name=restored', 'put', {spec: doc.spec, uid: '', resourceVersion: '', validateOnly: false});
});

it('refreshes an unchanged spec once after a status-only version conflict', async () => {
    (request as jest.Mock)
        .mockRejectedValueOnce({status: 409})
        .mockResolvedValueOnce({...doc, metadata: {...doc.metadata, resourceVersion: '43'}, status: {}})
        .mockResolvedValueOnce({...doc, spec: {parameterTemplate: 'next'}});
    const saved = jest.fn();
    render(<DefinitionEditor namespace='lab' resource='workflowforms' document={doc} onSaved={saved} />);
    fireEvent.click(screen.getByText('Edit definition spec'));
    fireEvent.change(screen.getByLabelText('Definition spec YAML'), {target: {value: 'parameterTemplate: next'}});
    fireEvent.click(screen.getByText('Save definition'));
    await waitFor(() => expect(saved).toHaveBeenCalledTimes(1));
    expect(request).toHaveBeenCalledTimes(3);
    expect(request).toHaveBeenLastCalledWith('lab', 'definitions/workflowforms?name=form', 'put', {
        spec: {parameterTemplate: 'next'},
        uid: 'original',
        resourceVersion: '43',
        validateOnly: false
    });
});

it.each([
    {...doc, spec: {parameterTemplate: 'other'}},
    {...doc, metadata: {...doc.metadata, uid: 'replacement'}},
    {...doc, metadata: {...doc.metadata, generation: 2}}
])('does not overwrite a replaced or concurrently edited definition', async changed => {
    (request as jest.Mock)
        .mockRejectedValueOnce({status: 409, message: 'definition changed'})
        .mockResolvedValueOnce({...changed, metadata: {...changed.metadata, resourceVersion: '43'}});
    const saved = jest.fn();
    render(<DefinitionEditor namespace='lab' resource='workflowforms' document={doc} onSaved={saved} />);
    fireEvent.click(screen.getByText('Edit definition spec'));
    fireEvent.change(screen.getByLabelText('Definition spec YAML'), {target: {value: 'parameterTemplate: next'}});
    fireEvent.click(screen.getByText('Save definition'));
    expect(await screen.findByRole('alert')).toHaveTextContent('definition changed');
    expect(request).toHaveBeenCalledTimes(2);
    expect(saved).not.toHaveBeenCalled();
    expect(screen.getByLabelText('Definition spec YAML')).toHaveValue('parameterTemplate: next');
});

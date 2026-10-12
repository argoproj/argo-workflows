import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import React from 'react';

import {request} from './api';
import {DefinitionPublications} from './definition-publications';

jest.mock('./api', () => ({request: jest.fn()}));
const doc = {metadata: {name: 'model', uid: 'source', resourceVersion: '10'}, spec: {schema: {type: 'object'}}};
beforeEach(() => (request as jest.Mock).mockReset());
it('publishes only the explicitly reviewed source and opens the returned immutable version', async () => {
    const saved = jest.fn();
    const published = {...doc, metadata: {...doc.metadata, name: 'model-v1', uid: 'published'}};
    (request as jest.Mock).mockResolvedValue(published);
    render(<DefinitionPublications namespace='lab' resource='parametertemplates' document={doc} onSelected={saved} />);
    expect(request).not.toHaveBeenCalled();
    fireEvent.click(screen.getByText('Publish named version'));
    fireEvent.change(screen.getByLabelText('Publication note'), {target: {value: 'first version'}});
    fireEvent.click(screen.getByText('Confirm publish definition'));
    await waitFor(() => expect(saved).toHaveBeenCalledWith(published));
    expect(request).toHaveBeenCalledWith('lab', 'definition-publications/parametertemplates', 'post', {
        source: 'model',
        uid: 'source',
        sourceSpec: doc.spec,
        name: 'model-v1',
        message: 'first version'
    });
});
it('loads a family page without definition bodies, compares on demand and rejects a replaced version', async () => {
    const saved = jest.fn();
    (request as jest.Mock)
        .mockResolvedValueOnce({items: [{metadata: {name: 'model-v1', uid: 'version-one'}}], metadata: {continue: 'next'}})
        .mockResolvedValueOnce({left: 'model-v1', right: 'model', changes: [{path: '/schema', op: 'replace'}], pagination: {total: 1}})
        .mockResolvedValueOnce({metadata: {name: 'model-v1', uid: 'replacement'}});
    render(<DefinitionPublications namespace='lab' resource='parametertemplates' document={doc} onSelected={saved} />);
    fireEvent.click(screen.getByText('View published versions'));
    expect(await screen.findByText('Load more definition versions')).toBeInTheDocument();
    fireEvent.click(screen.getByText('Compare model-v1 with selected definition'));
    expect(await screen.findByLabelText('Definition version comparison')).toHaveTextContent('/schema');
    fireEvent.click(screen.getByText('Open version model-v1'));
    expect(await screen.findByRole('alert')).toHaveTextContent('replaced');
    expect(saved).not.toHaveBeenCalled();
    expect(request).toHaveBeenNthCalledWith(1, 'lab', 'definition-publications/parametertemplates?family=source');
});
it('does not apply publication results after leaving a definition', async () => {
    let finish: (v: any) => void;
    (request as jest.Mock).mockReturnValue(
        new Promise(resolve => {
            finish = resolve;
        })
    );
    const saved = jest.fn();
    const view = render(<DefinitionPublications namespace='lab' resource='parametertemplates' document={doc} onSelected={saved} />);
    fireEvent.click(screen.getByText('Publish named version'));
    fireEvent.click(screen.getByText('Confirm publish definition'));
    view.unmount();
    finish(doc);
    await waitFor(() => expect(saved).not.toHaveBeenCalled());
});

import {act, fireEvent, render, screen, waitFor} from '@testing-library/react';
import React from 'react';

import {request} from './api';
import {OptionField} from './option-field';

jest.mock('./api', () => ({request: jest.fn()}));
beforeEach(() => (request as jest.Mock).mockReset());
it('waits for missing dependencies without fetching or dropping a saved selection', async () => {
    (request as jest.Mock).mockResolvedValue({items: [{value: 'east', label: 'East'}]});
    const context = {namespace: 'lab', binding: 'demo', sources: {'/region': {allowManual: false, dependsOn: ['/environment']}}, values: {}};
    const onChange = jest.fn();
    const view = render(<OptionField context={context} path='/region' value='east' onChange={onChange} disabled={false} />);
    expect(screen.getByText('Choose dependent values first: /environment')).toBeInTheDocument();
    expect(request).not.toHaveBeenCalled();
    expect(onChange).not.toHaveBeenCalled();
    view.rerender(<OptionField context={{...context, values: {environment: false}}} path='/region' value='east' onChange={onChange} disabled={false} />);
    await waitFor(() => expect(request).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.getByLabelText('/region')).toBeEnabled());
    expect(screen.getByLabelText('/region')).toHaveValue('"east"');
    view.rerender(<OptionField context={context} path='/region' value='east' onChange={onChange} disabled={false} />);
    expect(screen.getByLabelText('/region')).toBeDisabled();
    expect(screen.queryByRole('option', {name: 'East'})).not.toBeInTheDocument();
    expect(request).toHaveBeenCalledTimes(1);
    expect(onChange).not.toHaveBeenCalled();
});
it('binds option lookup to frozen input identity and ignores responses from prior values or nodes', async () => {
    const responses: Array<(v: any) => void> = [];
    (request as jest.Mock).mockImplementation(() => new Promise(resolve => responses.push(resolve)));
    const context = {
        namespace: 'lab',
        binding: '',
        sources: {'/region': {}},
        values: {environment: 'test'},
        input: {workflow: 'wf', uid: 'uid', nodeId: 'first', contractDigest: 'frozen'}
    };
    const onChange = jest.fn();
    const view = render(<OptionField context={context} path='/region' value='old' onChange={onChange} disabled={false} />);
    await waitFor(() => expect(responses).toHaveLength(1));
    expect(request).toHaveBeenLastCalledWith('lab', 'options', 'post', {input: context.input, path: '/region', values: {environment: 'test'}});
    view.rerender(<OptionField context={{...context, values: {environment: 'production'}}} path='/region' value='old' onChange={onChange} disabled={false} />);
    await waitFor(() => expect(responses).toHaveLength(2));
    view.rerender(
        <OptionField
            context={{...context, values: {environment: 'production'}, input: {...context.input, nodeId: 'second'}}}
            path='/region'
            value='old'
            onChange={onChange}
            disabled={false}
        />
    );
    await waitFor(() => expect(responses).toHaveLength(3));
    await act(async () => {
        responses[0]({items: [{value: 'a', label: 'Stale value'}]});
        responses[1]({items: [{value: 'b', label: 'Stale node'}]});
    });
    expect(screen.queryByText('Stale value')).not.toBeInTheDocument();
    expect(screen.queryByText('Stale node')).not.toBeInTheDocument();
    expect(screen.getByLabelText('/region')).toBeDisabled();
    await act(async () =>
        responses[2]({
            items: [
                {value: 'valid', label: 'Current option'},
                {value: 'denied', label: 'Unavailable option', disabled: true}
            ]
        })
    );
    expect(screen.getByLabelText('/region')).toBeEnabled();
    expect(screen.getByRole('option', {name: 'Unavailable option'})).toBeDisabled();
    expect(screen.getByRole('option', {name: 'Previous selection is unavailable'})).toBeDisabled();
    fireEvent.change(screen.getByLabelText('/region'), {target: {value: '"valid"'}});
    expect(onChange).toHaveBeenCalledWith('valid');
});

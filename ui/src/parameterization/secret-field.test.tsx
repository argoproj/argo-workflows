import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import React from 'react';

import {request} from './api';
import {SecretField} from './secret-field';

jest.mock('./api', () => ({request: jest.fn()}));
const context = {namespace: 'lab', template: 'typed', paths: ['/credential'], enabled: true};
beforeEach(() => {
    (request as jest.Mock).mockReset();
    Object.defineProperty(global, 'crypto', {value: {randomUUID: () => 'random-test-request-id'}, configurable: true});
});
it('does not put plaintext in ordinary form state and clears it after encrypted write', async () => {
    const change = jest.fn();
    const ref = {secretRef: {name: 'opaque-version', key: 'value'}};
    (request as jest.Mock).mockResolvedValue(ref);
    render(<SecretField context={context} path='/credential' value={undefined} onChange={change} disabled={false} />);
    fireEvent.change(screen.getByLabelText('New secret /credential'), {target: {value: 'synthetic-sensitive-input'}});
    expect(change).not.toHaveBeenCalled();
    fireEvent.click(screen.getByText('Store new encrypted secret version'));
    await waitFor(() => expect(change).toHaveBeenCalledWith(ref));
    expect(screen.getByLabelText('New secret /credential')).toHaveValue('');
});
it('never renders provider error text containing plaintext', async () => {
    (request as jest.Mock).mockRejectedValue(new Error('synthetic-sensitive-input'));
    render(<SecretField context={context} path='/credential' value={undefined} onChange={jest.fn()} disabled={false} />);
    fireEvent.change(screen.getByLabelText('New secret /credential'), {target: {value: 'synthetic-sensitive-input'}});
    fireEvent.click(screen.getByText('Store new encrypted secret version'));
    await screen.findByRole('alert');
    expect(screen.getByRole('alert')).not.toHaveTextContent('synthetic-sensitive-input');
});

it('discards a completed write after its field identity changes or unmounts', async () => {
    let finish: (value: any) => void;
    (request as jest.Mock).mockImplementation(
        () =>
            new Promise(resolve => {
                finish = resolve;
            })
    );
    const change = jest.fn();
    const busy = jest.fn();
    const {rerender, unmount} = render(<SecretField context={{...context, onBusyChange: busy}} path='/items/0/token' value={undefined} onChange={change} disabled={false} />);
    fireEvent.change(screen.getByLabelText('New secret /items/0/token'), {target: {value: 'synthetic-sensitive-input'}});
    fireEvent.click(screen.getByText('Store new encrypted secret version'));
    expect(busy).toHaveBeenCalledWith(true);
    rerender(<SecretField context={{...context, onBusyChange: busy}} path='/items/1/token' value={undefined} onChange={change} disabled={false} />);
    finish({secretRef: {name: 'late-result', key: 'value'}});
    await waitFor(() => expect(busy).toHaveBeenLastCalledWith(false));
    expect(change).not.toHaveBeenCalled();
    expect(screen.getByLabelText('New secret /items/1/token')).toHaveValue('');
    unmount();
});

it('edits a reference-only latest policy and requires an explicit policy switch', () => {
    const change = jest.fn();
    const view = render(<SecretField context={context} path='/credential' value={{secretRef: {name: 'fixed', key: 'token'}}} onChange={change} disabled={false} />);
    expect(screen.getByLabelText('Secret version policy /credential')).toHaveValue('fixed');
    fireEvent.change(screen.getByLabelText('Secret version policy /credential'), {target: {value: 'latest'}});
    expect(change).toHaveBeenLastCalledWith({latestSecretRef: {configMap: '', key: 'current'}});
    view.rerender(<SecretField context={context} path='/credential' value={{latestSecretRef: {configMap: 'versions', key: 'current'}}} onChange={change} disabled={false} />);
    expect(screen.getByText('Latest at submission: versions / current')).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText('Secret version index /credential'), {target: {value: 'rotated'}});
    expect(change).toHaveBeenLastCalledWith({latestSecretRef: {configMap: 'rotated', key: 'current'}});
    expect(request).not.toHaveBeenCalled();
    fireEvent.change(screen.getByLabelText('Secret version policy /credential'), {target: {value: 'fixed'}});
    expect(change).toHaveBeenLastCalledWith(undefined);
});

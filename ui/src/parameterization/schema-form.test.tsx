import {fireEvent, render, screen} from '@testing-library/react';
import React from 'react';

import {initialValue, SchemaForm} from './schema-form';

describe('structured input form', () => {
    it('does not move unsaved plaintext to another stable array identity', () => {
        const schema = {type: 'object', properties: {accounts: {type: 'array', items: {type: 'object', properties: {id: {type: 'string'}, credential: {type: 'object'}}}}}};
        function Editor() {
            const [value, change] = React.useState({accounts: [{id: 'one'}, {id: 'two'}]});
            return (
                <SchemaForm
                    schema={schema}
                    value={value}
                    onChange={change}
                    secretContext={{namespace: 'lab', template: 'model', paths: ['/accounts/*/credential'], enabled: true, identityRules: [{path: '/accounts', keyPath: '/id'}]}}
                />
            );
        }
        Object.defineProperty(global, 'crypto', {value: {randomUUID: () => 'random-test-id'}, configurable: true});
        render(<Editor />);
        fireEvent.change(screen.getByLabelText('New secret /accounts/0/credential'), {target: {value: 'synthetic-private-value'}});
        fireEvent.click(screen.getAllByText('Move up')[1]);
        expect(screen.getByLabelText('/accounts/0/id')).toHaveValue('two');
        expect(screen.getByLabelText('New secret /accounts/0/credential')).toHaveValue('');
        expect(screen.getByLabelText('New secret /accounts/1/credential')).toHaveValue('');
    });
    it('preserves false, zero and null defaults', () => {
        expect(initialValue({default: false})).toBe(false);
        expect(initialValue({default: 0})).toBe(0);
        expect(initialValue({default: null})).toBeNull();
    });
    it('distinguishes unset from an explicit false', () => {
        const change = jest.fn();
        render(<SchemaForm schema={{type: 'boolean'}} path='/enabled' value={false} onChange={change} />);
        expect(screen.getByLabelText('/enabled')).toHaveValue('false');
        fireEvent.change(screen.getByLabelText('/enabled'), {target: {value: ''}});
        expect(change).toHaveBeenLastCalledWith(undefined);
    });
    it('edits arrays without retaining a removed item', () => {
        const change = jest.fn();
        render(<SchemaForm schema={{type: 'array', items: {type: 'string'}}} path='/zones' value={['north', 'south']} onChange={change} />);
        fireEvent.click(screen.getByLabelText('Remove /zones/0'));
        expect(change).toHaveBeenLastCalledWith(['south']);
    });
    it('escapes object pointers and respects disabled input', () => {
        render(<SchemaForm schema={{type: 'object', properties: {'a/b~c': {type: 'string'}}}} value={{'a/b~c': 'value'}} onChange={jest.fn()} disabled={true} />);
        expect(screen.getByLabelText('/a~1b~0c')).toBeDisabled();
    });
});

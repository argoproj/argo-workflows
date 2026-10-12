import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import React, {useState} from 'react';

import {DeclaredForm} from './declared-form';
import {setAt, validateForm} from './form-definition';

const schema = {
    type: 'object',
    additionalProperties: false,
    properties: {
        'enabled': {type: 'boolean'},
        'region': {type: 'string', enum: ['east', 'west']},
        'settings': {type: 'object', properties: {count: {type: 'integer'}, comment: {type: 'string'}}},
        'zones': {type: 'array', items: {type: 'object', properties: {name: {type: 'string'}, active: {type: 'boolean'}, count: {type: 'integer'}}}},
        'extra': {type: 'object', properties: {count: {type: 'integer'}}},
        'a/b~c': {type: 'string'}
    }
};
const form: any = {
    layout: [
        {id: 'basic', kind: 'tab'},
        {id: 'network', kind: 'tab'},
        {id: 'nested', parent: 'network', kind: 'tree', label: 'Nested zones'}
    ],
    fields: [
        {path: '/enabled', widget: 'checkbox', label: 'Enable changes', group: 'basic'},
        {path: '/region', widget: 'select', group: 'basic'},
        {path: '/settings', group: 'basic', enabledWhen: {path: '/enabled', operator: 'Equals', value: true}},
        {path: '/settings/comment', visibleWhen: {path: '/region', operator: 'Equals', value: 'west'}},
        {path: '/zones', widget: 'table', group: 'nested'},
        {path: '/zones/*/active', widget: 'checkbox'},
        {path: '/zones/*/count', enabledWhen: {path: '/zones/*/active', operator: 'Equals', value: true}},
        {path: '/extra', widget: 'yaml', group: 'network'},
        {path: '/a~1b~0c', readOnly: true}
    ]
};
function Editor({definition = form}: {definition?: any}) {
    const [value, setValue] = useState({
        'enabled': false,
        'region': 'east',
        'settings': {count: 0, comment: 'retain'},
        'zones': [
            {name: 'a', active: false, count: 0},
            {name: 'b', active: true, count: 2}
        ],
        'extra': {count: 1},
        'a/b~c': 'immutable'
    });
    const [valid, setValid] = useState(true);
    return (
        <>
            <DeclaredForm schema={schema} value={value} onChange={setValue} form={definition} onValidityChange={setValid} />
            <button disabled={!valid}>Save</button>
            <output aria-label='values'>{JSON.stringify(value)}</output>
        </>
    );
}
it('keeps hidden values, enables nested controls and renders grouped fields once', () => {
    render(<Editor />);
    expect(screen.getByLabelText('/settings/count')).toBeDisabled();
    expect(screen.queryByLabelText('/settings/comment')).toBeNull();
    expect(screen.getByLabelText('values')).toHaveTextContent('retain');
    fireEvent.click(screen.getByLabelText('/enabled'));
    expect(screen.getAllByLabelText('/settings/count')).toHaveLength(1);
    expect(screen.getByLabelText('/settings/count')).not.toBeDisabled();
    fireEvent.change(screen.getByLabelText('/region'), {target: {value: '"west"'}});
    expect(screen.getByLabelText('/settings/comment')).toHaveValue('retain');
    expect(screen.getByLabelText('/a~1b~0c')).toBeDisabled();
});
it('renders table rows, resolves wildcard conditions per row, and supports keyboard tabs', () => {
    render(<Editor />);
    fireEvent.keyDown(screen.getByRole('tab', {name: 'basic'}), {key: 'ArrowRight'});
    expect(screen.getByRole('tab', {name: 'network'})).toHaveAttribute('aria-selected', 'true');
    expect(screen.getByLabelText('/zones/0/count')).toBeDisabled();
    expect(screen.getByLabelText('/zones/1/count')).not.toBeDisabled();
    fireEvent.click(screen.getByLabelText('/zones/0/active'));
    fireEvent.change(screen.getByLabelText('/zones/0/count'), {target: {value: '3'}});
    fireEvent.click(screen.getByLabelText('Remove /zones/1'));
    expect(screen.queryByLabelText('/zones/1/count')).toBeNull();
    expect(screen.getByLabelText('values')).toHaveTextContent('"count":3');
});
it('blocks save for invalid widget YAML even when moving to another tab', async () => {
    render(<Editor />);
    fireEvent.click(screen.getByRole('tab', {name: 'network'}));
    fireEvent.change(screen.getByLabelText('/extra'), {target: {value: 'count: ['}});
    await waitFor(() => expect(screen.getByRole('button', {name: 'Save'})).toBeDisabled());
    fireEvent.click(screen.getByRole('tab', {name: 'basic'}));
    expect(screen.getByRole('button', {name: 'Save'})).toBeDisabled();
    fireEvent.click(screen.getByRole('tab', {name: 'network'}));
    expect(screen.getByLabelText('/extra')).toHaveValue('count: [');
    fireEvent.change(screen.getByLabelText('/extra'), {target: {value: 'count: 0'}});
    await waitFor(() => expect(screen.getByRole('button', {name: 'Save'})).not.toBeDisabled());
    expect(screen.getByLabelText('values')).toHaveTextContent('"extra":{"count":0}');
});
it('rejects malformed layout, mismatched widgets, wrong paths and unbound conditions', () => {
    expect(validateForm({options: {'/missing': {}}}, schema)).toContain('Invalid option field: /missing');
    expect(validateForm({options: {'/region': {dependsOn: ['/missing']}}}, schema)).toContain('Invalid option dependency: /missing');
    expect(validateForm({options: {'/region': {dependsOn: ['/a~1b~0c']}}}, schema)).toEqual([]);
    expect(
        validateForm(
            {
                layout: [
                    {id: 'a', parent: 'b', kind: 'tree'},
                    {id: 'b', parent: 'a', kind: 'tree'}
                ]
            },
            schema
        ).join()
    ).toMatch(/cycle/);
    for (const field of [
        {path: '/missing'},
        {path: '/enabled', widget: 'number'},
        {path: '/settings', group: 'missing'},
        {path: '/region', enabledWhen: {path: '/zones/*/active', operator: 'Exists'}},
        {path: '/zones/0/count', group: 'group'}
    ]) {
        expect(validateForm({layout: [{id: 'group', kind: 'group'}], fields: [field as any]}, schema).length).toBeGreaterThan(0);
    }
    render(<Editor definition={{fields: [{path: '/region', widget: 'secret'}]}} />);
    expect(screen.getByRole('alert')).toHaveTextContent('declared sensitive');
});
it('updates escaped and prototype-named keys without prototype mutation', () => {
    const value = setAt({}, '/__proto__/safe', 0);
    expect(Object.getPrototypeOf(value)).toBe(Object.prototype);
    expect(({} as any).safe).toBeUndefined();
    expect(JSON.stringify(value)).toBe('{"__proto__":{"safe":0}}');
    expect(setAt({'a/b': {'~x': 1}}, '/a~1b/~0x', null)).toEqual({'a/b': {'~x': null}});
});

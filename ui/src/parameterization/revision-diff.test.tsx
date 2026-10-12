import {fireEvent, render, screen} from '@testing-library/react';
import * as React from 'react';

import {RevisionDiff} from './revision-diff';

it('distinguishes absent/null, false/zero, type changes, and stable item moves', () => {
    render(
        <RevisionDiff
            comparison={{
                left: 'r1',
                right: 'r2',
                changes: [
                    {op: 'add', path: '/optional', before: null, after: null},
                    {op: 'replace', path: '/enabled', before: true, after: false},
                    {op: 'type', path: '/count', before: '0', after: 0},
                    {op: 'move', path: '/zones/0', from: '/zones/1', itemIdentity: 'west', before: null, after: null}
                ]
            }}
        />
    );
    expect(screen.getByText('null')).toBeVisible();
    expect(screen.getByText('false')).toBeVisible();
    expect(screen.getByText('"0"')).toBeVisible();
    expect(screen.getByText('0')).toBeVisible();
    expect(screen.getByText('Type changed')).toBeVisible();
    expect(screen.getByText('west')).toBeVisible();
    expect(screen.getAllByText('/zones/1')).toHaveLength(2);
    fireEvent.change(screen.getByLabelText('Filter revision changes'), {target: {value: 'west'}});
    expect(screen.queryByText('/optional')).toBeNull();
    expect(screen.getAllByText('/zones/0')).toHaveLength(2);
});
it('separates model/source changes and explains changed array identity rules', () => {
    render(
        <RevisionDiff
            comparison={{
                modelChanged: true,
                identityRulesChanged: true,
                modelChanges: [{op: 'replace', path: '/schema/properties/count/default', before: 1, after: 2}],
                sourceChanges: [{op: 'replace', path: '/count', before: 'default', after: 'user'}]
            }}
        />
    );
    expect(screen.getByText('Parameter model (1/1)')).toBeVisible();
    expect(screen.getByText('Value sources (1/1)')).toBeVisible();
    expect(screen.getByText(/Array identity rules changed/)).toBeVisible();
    expect(screen.getByText('"user"')).toBeVisible();
});
it('shows secret reference metadata while hiding malformed sensitive values', () => {
    const view = render(
        <RevisionDiff
            comparison={{
                sensitivePaths: ['/credential'],
                changes: [{op: 'replace', path: '/credential', before: 'do-not-render', after: {secretRef: {name: 'version-2', key: 'value', version: '2'}}}]
            }}
        />
    );
    expect(screen.queryByText(/do-not-render/)).toBeNull();
    expect(screen.getByText(/Secret reference:.*version-2/)).toBeVisible();
    view.rerender(
        <RevisionDiff comparison={{sensitivePaths: ['/credential'], changes: [{op: 'replace', path: '/credential/secretRef/name', before: 'version-1', after: 'version-2'}]}} />
    );
    expect(screen.getByText('"version-2"')).toBeVisible();
});

it('does not expose a nested secret when its entire parent is replaced', () => {
    render(<RevisionDiff comparison={{sensitivePaths: ['/options/password'], changes: [{op: 'replace', path: '/options', before: null, after: {password: 'never-display'}}]}} />);
    expect(screen.queryByText(/never-display/)).toBeNull();
    expect(screen.getAllByText(/Contains sensitive fields/)).toHaveLength(2);
});

it('separates execution layers and identifies unavailable history instead of claiming no changes', () => {
    render(
        <RevisionDiff
            comparison={{
                layers: [
                    {name: 'Parameter binding', available: false, notice: 'Historical definition content was not recorded.', changes: []},
                    {name: 'Compiled inputs', available: true, changes: [{op: 'replace', path: '/workflow/count', before: '2', after: '4'}]}
                ]
            }}
        />
    );
    expect(screen.getByText('Parameter binding (0/0)')).toBeVisible();
    expect(screen.getByText('Historical definition content was not recorded.')).toBeVisible();
    expect(screen.getByText('Compiled inputs (1/1)')).toBeVisible();
    expect(screen.getByText('"4"')).toBeVisible();
});

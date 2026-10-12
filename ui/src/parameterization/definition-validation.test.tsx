import {render, screen} from '@testing-library/react';
import * as React from 'react';

import {DefinitionValidation} from './definition-validation';

test('does not present unobserved or stale conditions as valid', () => {
    const document: any = {metadata: {uid: 'one', generation: 2}};
    const {rerender} = render(<DefinitionValidation document={document} />);
    expect(screen.getByText('Not observed')).toBeTruthy();
    document.status = {observedGeneration: 1, conditions: [{type: 'DefinitionValid', status: 'True', message: 'old success'}]};
    rerender(<DefinitionValidation document={document} />);
    expect(screen.getByText('Validation pending for this generation')).toBeTruthy();
    expect(screen.queryByText('old success')).toBeNull();
});

test('reports errors, references and advisory nature of static checks', () => {
    const document = {
        metadata: {uid: 'one', generation: 2},
        status: {
            observedGeneration: 2,
            conditions: [{type: 'DefinitionValid', status: 'False', message: 'check failed'}],
            errors: [{code: 'ReferenceUnavailable', path: '/spec', message: 'model absent'}],
            resolvedReferences: [{kind: 'WorkflowForm', name: 'form', uid: 'form-one', digest: 'abcd1234'}]
        }
    };
    render(<DefinitionValidation document={document} />);
    expect(screen.getByText('Definition invalid')).toBeTruthy();
    expect(screen.getByText(/ReferenceUnavailable/)).toBeTruthy();
    expect(screen.getByText(/do not grant execution permission/)).toBeTruthy();
    expect(screen.getByText('WorkflowForm/form')).toBeTruthy();
});

import {fireEvent, render} from '@testing-library/react';
import React, {forwardRef} from 'react';

import {ObjectEditor} from './object-editor';

// Mock the heavy Monaco editor
jest.mock('./suspense-monaco-editor', () => ({
    // eslint-disable-next-line react/display-name, @typescript-eslint/no-unused-vars
    SuspenseMonacoEditor: forwardRef((props: any, _) => <textarea data-testid='editor' onChange={e => props.onChange(e.target.value)} />)
}));

describe('ObjectEditor', () => {
    const value = {foo: 'bar', baz: 123};
    const text = JSON.stringify(value, null, 2);

    it('calls onLangChange with the opposite language when toggle is clicked', () => {
        const onLangChange = jest.fn();
        const onChange = jest.fn();

        const {getByRole} = render(<ObjectEditor value={value} text={text} lang='json' onLangChange={onLangChange} onChange={onChange} />);

        const toggle = getByRole('button', {name: /JSON\s*\/\s*YAML/});
        expect(toggle).toBeInTheDocument();

        toggle.click();
        expect(onLangChange).toHaveBeenCalledWith('yaml');
    });

    it('renders a navigation button for each key in the object', () => {
        const onLangChange = jest.fn();
        const onChange = jest.fn();

        const {getByRole} = render(<ObjectEditor value={value} text={text} lang='json' onLangChange={onLangChange} onChange={onChange} />);

        expect(getByRole('button', {name: 'foo'})).toBeInTheDocument();
        expect(getByRole('button', {name: 'baz'})).toBeInTheDocument();
    });

    it('calls onChange when editor text changes', () => {
        const onLangChange = jest.fn();
        const onChange = jest.fn();

        const {getByTestId} = render(<ObjectEditor value={value} text={text} lang='json' onLangChange={onLangChange} onChange={onChange} />);

        const editor = getByTestId('editor') as HTMLTextAreaElement;
        const updated = JSON.stringify({...value, foo: 'baz'}, null, 2);

        fireEvent.change(editor, {target: {value: updated}});

        expect(onChange).toHaveBeenCalledWith(updated);
    });
});

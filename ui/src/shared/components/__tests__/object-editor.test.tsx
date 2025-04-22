import {fireEvent, render} from '@testing-library/react';
import React, {forwardRef, useImperativeHandle} from 'react';

import {ObjectEditor} from '../object-editor';

// Mock the heavy Monaco editor
jest.mock('../suspense-monaco-editor', () => {
    return {
        // eslint-disable-next-line react/display-name
        SuspenseMonacoEditor: forwardRef((props: any, ref: any) => {
            const {defaultValue, ...forward} = props;

            useImperativeHandle(ref, () => ({
                editor: {
                    getValue: () => defaultValue,
                    setValue: () => {},
                    revealLineInCenter: () => {},
                    setPosition: () => {},
                    focus: () => {}
                }
            }));

            /* Render a textarea so fireEvent.change works with a native `value`
         property.  Spread `forward` *first* so we can wrap its onChange
         and pass only the updated string back to parent code. */
            return <textarea data-testid='editor' defaultValue={defaultValue} {...forward} onChange={e => forward.onChange?.(e.target.value)} />;
        })
    };
});

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

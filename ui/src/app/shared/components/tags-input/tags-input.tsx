import classNames from 'classnames';
import * as React from 'react';
import {useEffect, useRef, useState} from 'react';

import {Autocomplete, AutocompleteApi, AutocompleteOption} from 'argo-ui';

interface TagsInputProps {
    tags: string[];
    autocomplete?: (AutocompleteOption | string)[];
    sublistQuery?: (key: string) => Promise<any>;
    onChange?: (tags: string[]) => void;
    placeholder?: string;
}

require('./tags-input.scss');

export function TagsInput(props: TagsInputProps) {
    const inputRef = useRef<HTMLInputElement>(null);
    const autoCompleteRef = useRef<AutocompleteApi>(null);

    const [tags, setTags] = useState(props.tags || []);
    const [input, setInput] = useState('');
    const [focused, setFocused] = useState(false);
    const [subTags, setSubTags] = useState<string[]>([]);
    const [subTagsActive, setSubTagsActive] = useState(false);

    useEffect(() => {
        if (props.onChange) {
            props.onChange(tags);
            setTimeout(() => autoCompleteRef.current?.refresh());
        }
    }, [tags]);

    return (
        <div className={classNames('tags-input argo-field', {'tags-input--focused': focused || !!input})} onClick={() => inputRef.current?.focus()}>
            {props.tags ? (
                props.tags.map((tag, i) => (
                    <span className='tags-input__tag' key={tag}>
                        {tag}{' '}
                        <i
                            className='fa fa-times'
                            onClick={e => {
                                const newTags = [...tags.slice(0, i), ...tags.slice(i + 1)];
                                setTags(newTags);
                                e.stopPropagation();
                            }}
                        />
                    </span>
                ))
            ) : (
                <span />
            )}
            <Autocomplete
                filterSuggestions={true}
                autoCompleteRef={ref => (autoCompleteRef.current = ref)}
                value={input}
                items={subTagsActive ? subTags : props.autocomplete}
                onChange={e => setInput(e.target.value)}
                onSelect={async value => {
                    if (props.sublistQuery != null && !subTagsActive) {
                        setSubTagsActive(true);
                        const newSubTags = await props.sublistQuery(value);
                        setSubTags(newSubTags || []);
                    } else {
                        if (tags.indexOf(value) === -1) {
                            const newTags = tags.concat(value);
                            setTags(newTags);
                            setInput('');
                            setSubTags([]);
                        }
                        setSubTagsActive(false);
                    }
                }}
                renderInput={inputProps => (
                    <input
                        {...inputProps}
                        placeholder={props.placeholder}
                        ref={ref => {
                            inputRef.current = ref;
                            if (typeof inputProps.ref === 'function') {
                                inputProps.ref(ref);
                            }
                        }}
                        onFocus={e => {
                            inputProps.onFocus?.(e);
                            setFocused(true);
                        }}
                        onBlur={e => {
                            inputProps.onBlur?.(e);
                            setFocused(false);
                        }}
                        onKeyDown={e => {
                            if (e.keyCode === 8 && tags.length > 0 && input === '') {
                                const newTags = tags.slice(0, tags.length - 1);
                                setTags(newTags);
                            }
                            inputProps.onKeyDown?.(e);
                        }}
                        onKeyUp={e => {
                            if (e.keyCode === 13 && input && tags.indexOf(input) === -1) {
                                const newTags = tags.concat(input);
                                setTags(newTags);
                                setInput('');
                            }
                            inputProps.onKeyUp?.(e);
                        }}
                    />
                )}
            />
        </div>
    );
}

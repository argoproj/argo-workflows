import {Autocomplete, AutocompleteApi, AutocompleteOption} from 'argo-ui';
import classNames from 'classnames';
import * as React from 'react';
import {useRef, useState} from 'react';

import './tags-input.scss';

interface TagsInputProps {
    tags: string[];
    autocomplete?: (AutocompleteOption | string)[];
    sublistQuery?: (key: string) => Promise<any>;
    onChange: (tags: string[]) => void;
    placeholder?: string;
}

export function TagsInput(props: TagsInputProps) {
    const inputRef = useRef<HTMLInputElement>(null);
    const autoCompleteRef = useRef<AutocompleteApi>(null);

    const [input, setInput] = useState('');
    const [focused, setFocused] = useState(false);
    const [subTags, setSubTags] = useState<string[]>([]);
    const [subTagsActive, setSubTagsActive] = useState(false);

    function setTags(tags: string[]) {
        props.onChange(tags);
        setTimeout(() => autoCompleteRef.current?.refresh());
    }

    return (
        <div className={classNames('tags-input argo-field', {'tags-input--focused': focused || !!input})} onClick={() => inputRef.current?.focus()}>
            {props.tags ? (
                props.tags.map((tag, i) => (
                    <span className='tags-input__tag' key={tag}>
                        {tag}{' '}
                        <i
                            className='fa fa-times'
                            onClick={e => {
                                setTags([...props.tags.slice(0, i), ...props.tags.slice(i + 1)]);
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
                        if (props.tags.indexOf(value) === -1) {
                            setInput('');
                            setSubTags([]);
                            setTags(props.tags.concat(value));
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
                            if (e.keyCode === 8 && props.tags.length > 0 && input === '') {
                                setTags(props.tags.slice(0, props.tags.length - 1));
                            }
                            inputProps.onKeyDown?.(e);
                        }}
                        onKeyUp={e => {
                            if (e.keyCode === 13 && input && props.tags.indexOf(input) === -1) {
                                setInput('');
                                setTags(props.tags.concat(input));
                            }
                            inputProps.onKeyUp?.(e);
                        }}
                    />
                )}
            />
        </div>
    );
}

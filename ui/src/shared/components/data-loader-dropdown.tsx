import {DataLoader} from 'argo-ui/src/components/data-loader';
import {Select, SelectOption} from 'argo-ui/src/components/select/select';
import * as React from 'react';
import {useEffect, useRef, useState} from 'react';

export function DataLoaderDropdown(props: {load: () => Promise<(string | SelectOption)[]>; onChange: (value: string) => void; placeholder?: string}) {
    const [selected, setSelected] = useState('');
    const selectRef = useRef<HTMLDivElement>(null);
    const [options, setOptions] = useState<(string | SelectOption)[]>([]);

    const enhanceOptionsWithTitles = (options: (string | SelectOption)[]) => {
        setOptions(options);
        return options.map(option => {
            if (typeof option === 'string') {
                return {
                    value: option,
                    title: option
                };
            } else {
                return option;
            }
        });
    };

    useEffect(() => {
        const addTitleAttributes = () => {
            setTimeout(() => {
                const optionElements = document.querySelectorAll('.select__option');
                optionElements.forEach(element => {
                    const text = element.textContent || '';
                    if (text) {
                        element.setAttribute('title', text);
                    }
                });
            }, 100);
        };

        if (options.length > 0) {
            addTitleAttributes();
        }
    }, [options]);

    useEffect(() => {
        const handleDropdownOpen = () => {
            const observer = new MutationObserver(mutations => {
                mutations.forEach(mutation => {
                    if (mutation.type === 'childList' && mutation.addedNodes.length > 0) {
                        const optionElements = document.querySelectorAll('.select__option');
                        optionElements.forEach(element => {
                            const text = element.textContent || '';
                            if (text) {
                                element.setAttribute('title', text);
                            }
                        });
                    }
                });
            });

            observer.observe(document.body, {childList: true, subtree: true});

            return () => {
                observer.disconnect();
            };
        };

        handleDropdownOpen();
    }, []);

    return (
        <div ref={selectRef}>
            <DataLoader load={props.load}>
                {list => (
                    <Select
                        placeholder={props.placeholder}
                        options={enhanceOptionsWithTitles(list)}
                        value={selected}
                        onChange={x => {
                            setSelected(x.value);
                            props.onChange(x.value);
                        }}
                    />
                )}
            </DataLoader>
        </div>
    );
}

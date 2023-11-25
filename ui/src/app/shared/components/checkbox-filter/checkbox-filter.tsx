import {Checkbox} from 'argo-ui';
import * as React from 'react';

import './checkbox-filter.scss';

interface Props {
    items: {name: string; count: number}[];
    type: string;
    selected: string[];
    onChange: (selected: string[]) => void;
}

export function CheckboxFilter(props: Props) {
    const unavailableSelected = props.selected.filter(selected => !props.items.some(item => item.name === selected));
    const items = props.items.concat(unavailableSelected.map(selected => ({name: selected, count: 0})));

    return (
        <ul className='checkbox-filter columns small-12'>
            {items.map(item => (
                <li key={item.name}>
                    <React.Fragment>
                        <div className='row'>
                            <div className='checkbox-filter__label columns small-12'>
                                <Checkbox
                                    checked={props.selected.indexOf(item.name) > -1}
                                    id={`filter-${props.type}-${item.name}`}
                                    onChange={() => {
                                        const newSelected = props.selected.slice();
                                        const index = newSelected.indexOf(item.name);
                                        if (index > -1) {
                                            newSelected.splice(index, 1);
                                        } else {
                                            newSelected.push(item.name);
                                        }
                                        props.onChange(newSelected);
                                    }}
                                />{' '}
                                <label title={item.name} htmlFor={`filter-${props.type}-${item.name}`}>
                                    {item.name}
                                </label>
                            </div>
                        </div>
                    </React.Fragment>
                </li>
            ))}
        </ul>
    );
}

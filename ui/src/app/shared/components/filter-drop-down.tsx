import {Checkbox} from 'argo-ui';
import classNames from 'classnames';
import * as React from 'react';
import {DropDown} from './dropdown/dropdown';

interface FilterDropDownProps {
    sections: FilterDropSection[];
}

export interface FilterDropSection {
    title: string;
    values: {[label: string]: boolean};
    onChange: (label: string, checked: boolean) => void;
}

export const FilterDropDown = (props: FilterDropDownProps) => {
    return (
        <DropDown
            isMenu={true}
            anchor={
                <div className={classNames('top-bar__filter')} title='Filter'>
                    <i className='argo-icon-filter' aria-hidden='true' />
                    <i className='fa fa-angle-down' aria-hidden='true' />
                </div>
            }>
            <ul>
                {props.sections
                    .filter(item => item.values)
                    .map((item, i) => (
                        <div key={i}>
                            <li key={i} className={classNames('top-bar__filter-item', {title: true})}>
                                <span>{item.title}</span>
                            </li>
                            {Object.entries(item.values)
                                .sort()
                                .map(([label, checked]) => (
                                    <li key={label} className={classNames('top-bar__filter-item')}>
                                        <React.Fragment>
                                            <Checkbox id={`filter__${i}_${label}`} checked={checked} onChange={v => item.onChange(label, v)} />
                                            <label htmlFor={`filter__${i}_${label}`}>{label}</label>
                                        </React.Fragment>
                                    </li>
                                ))}
                        </div>
                    ))}
            </ul>
        </DropDown>
    );
};

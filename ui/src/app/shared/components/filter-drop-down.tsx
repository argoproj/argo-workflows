import {DropDown} from 'argo-ui/src/components/dropdown/dropdown';
import * as React from 'react';
import {CheckboxList} from './checkbox-list';

export const FilterDropDown = (props: {values: {[label: string]: boolean}; onChange: (label: string, checked: boolean) => void}) => (
    <DropDown
        isMenu={true}
        anchor={() => (
            <div className={'top-bar__filter' + (props.values.size > props.values.size ? ' top-bar__filter--selected' : '')}>
                <i className='argo-icon-filter' />
                <i className='fa fa-angle-down' />
            </div>
        )}>
        <CheckboxList values={props.values} onChange={(label, checked) => props.onChange(label, checked)} />
    </DropDown>
);

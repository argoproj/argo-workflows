import * as React from 'react';
import {Checkbox} from 'argo-ui';

require('./checkbox-filter.scss');

export class CheckboxFilter extends React.Component<
    {
        items: {name: string; count: number}[];
        type: string;
        selected: string[];
        onChange: (selected: string[]) => any;
    },
    {}
> {
    constructor(props: any) {
        super(props);
    }

    public render() {
        const unavailableSelected = this.props.selected.filter(selected => !this.props.items.some(item => item.name === selected));
        const items = this.props.items.concat(unavailableSelected.map(selected => ({name: selected, count: 0})));
        return (
            <ul className='checkbox__filter'>
                {items.map(item => (
                    <li key={item.name}>
                        <React.Fragment>
                            <div className='checkbox__filter-label'>
                                <Checkbox
                                    checked={this.props.selected.indexOf(item.name) > -1}
                                    id={`filter-${this.props.type}-${item.name}`}
                                    onChange={() => {
                                        const newSelected = this.props.selected.slice();
                                        const index = newSelected.indexOf(item.name);
                                        if (index > -1) {
                                            newSelected.splice(index, 1);
                                        } else {
                                            newSelected.push(item.name);
                                        }
                                        this.props.onChange(newSelected);
                                    }}
                                />{' '}
                                <label title={item.name} htmlFor={`filter-${this.props.type}-${item.name}`}>
                                    {item.name}
                                </label>
                            </div>{' '}
                            <span>{item.count}</span>
                        </React.Fragment>
                    </li>
                ))}
            </ul>
        );
    }
}

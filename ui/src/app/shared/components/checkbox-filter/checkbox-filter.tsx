import {Checkbox} from 'argo-ui';
import * as React from 'react';

require('./checkbox-filter.scss');

interface Props {
    items: {name: string; count: number}[];
    type: string;
    selected: string[];
    onChange: (selected: string[]) => void;
}

export class CheckboxFilter extends React.Component<Props> {
    constructor(props: any) {
        super(props);
    }

    public render() {
        const unavailableSelected = this.props.selected.filter(selected => !this.props.items.some(item => item.name === selected));
        const items = this.props.items.concat(unavailableSelected.map(selected => ({name: selected, count: 0})));
        return (
            <ul className='checkbox-filter columns small-12'>
                {items.map(item => (
                    <li key={item.name}>
                        <React.Fragment>
                            <div className='row'>
                                <div className='checkbox-filter__label columns small-12'>
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
                                </div>
                            </div>
                        </React.Fragment>
                    </li>
                ))}
            </ul>
        );
    }
}

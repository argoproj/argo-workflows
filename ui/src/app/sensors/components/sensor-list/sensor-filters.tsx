import * as React from 'react';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';

interface Props {
    namespace: string;
    onChange: (namespace: string) => void;
}

export class SensorFilters extends React.Component<Props, {}> {
    public render() {
        return (
            <div className='wf-filters-container'>
                <div className='row'>
                    <div className='columns small-3 xlarge-12'>
                        <p className='wf-filters-container__title'>Namespace</p>
                        <NamespaceFilter value={this.props.namespace} onChange={this.props.onChange} />
                    </div>
                </div>
            </div>
        );
    }
}

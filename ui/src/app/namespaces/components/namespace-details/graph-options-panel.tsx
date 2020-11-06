import * as classNames from 'classnames';
import * as React from 'react';

require('./graph-options-panel.scss');

interface GraphOptions {
    markActivations: boolean;
}

export class GraphOptionsPanel extends React.Component<GraphOptions & {onChange: (changed: GraphOptions) => void}> {
    public render() {
        return (
            <div className='graph-options-panel'>
                <a
                    className={classNames({active: this.props.markActivations})}
                    onClick={() => this.props.onChange({...this.props, markActivations: !this.props.markActivations})}
                    title='Mark entities when they activate'>
                    <i className={classNames('fa', 'fa-circle-notch', {'fa-spin': this.props.markActivations})} />
                </a>
            </div>
        );
    }
}

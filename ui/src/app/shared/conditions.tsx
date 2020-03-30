import * as React from 'react';
import {WarningWorkflowConditions, WorkflowCondition} from '../../models';

interface Props {
    conditions: WorkflowCondition[];
}

export class Conditions extends React.Component<Props> {
    public render() {
        return (
            <>
                {this.props.conditions &&
                    Object.entries(this.props.conditions).map(([_, condition]) => {
                        return (
                            <div key={condition.type} style={{lineHeight: '120%', marginTop: '16px'}}>
                                {WarningWorkflowConditions.includes(condition.type) && <span className={'fa fa-exclamation-triangle'} style={{color: '#d7b700'}} />}{' '}
                                {condition.type + ': ' + (condition.message || condition.status)}
                            </div>
                        );
                    })}
            </>
        );
    }
}

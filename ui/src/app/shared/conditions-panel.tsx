import * as React from 'react';
import {WorkflowCondition, WorkflowConditionType} from '../../models';

interface Props {
    conditions: WorkflowCondition[];
}

const WarningWorkflowConditions: WorkflowConditionType[] = ['SpecWarning'];

export function hasWarningCondition(conditions: WorkflowCondition[]): boolean {
    if (conditions.length === 0) {
        return false;
    }

    for (const condition of conditions) {
        if (WarningWorkflowConditions.includes(condition.type)) {
            return true;
        }
    }

    return false;
}

export class ConditionsPanel extends React.Component<Props> {
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

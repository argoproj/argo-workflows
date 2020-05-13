import * as React from 'react';
import {WorkflowCondition, WorkflowConditionType} from '../../models';

interface Props {
    conditions: WorkflowCondition[];
}

const WarningWorkflowConditions: WorkflowConditionType[] = ['SpecWarning'];
const ErrorWorkflowConditions: WorkflowConditionType[] = ['MetricsError'];

export function hasWarningConditionBadge(conditions: WorkflowCondition[]): boolean {
    if (conditions.length === 0) {
        return false;
    }

    for (const condition of conditions) {
        if (WarningWorkflowConditions.includes(condition.type)) {
            return true;
        }
        if (ErrorWorkflowConditions.includes(condition.type)) {
            return true;
        }
    }

    return false;
}

function getConditionIcon(condition: WorkflowConditionType): JSX.Element {
    if (WarningWorkflowConditions.includes(condition)) {
        return <span className={'fa fa-exclamation-triangle'} style={{color: '#d7b700'}} />;
    }
    if (ErrorWorkflowConditions.includes(condition)) {
        return <span className={'fa fa-exclamation-circle'} style={{color: '#d70022'}} />;
    }
    return <span />;
}

export class ConditionsPanel extends React.Component<Props> {
    public render() {
        return (
            <>
                {this.props.conditions &&
                    Object.entries(this.props.conditions).map(([_, condition]) => {
                        return (
                            <div key={condition.type} style={{lineHeight: '120%', marginTop: '16px'}}>
                                {getConditionIcon(condition.type)} {condition.type + ': ' + (condition.message || condition.status)}
                            </div>
                        );
                    })}
            </>
        );
    }
}

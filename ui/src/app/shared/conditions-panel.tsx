import * as React from 'react';
import {CronWorkflowCondition, CronWorkflowConditionType, WorkflowCondition, WorkflowConditionType} from '../../models';

interface Props {
    conditions: WorkflowCondition[] | CronWorkflowCondition[];
}

const WarningWorkflowConditions: WorkflowConditionType[] = ['SpecWarning'];
const ErrorWorkflowConditions: WorkflowConditionType[] = ['MetricsError'];
const ErrorCronWorkflowConditions: CronWorkflowConditionType[] = ['SubmissionError'];

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

function getConditionIcon(condition: WorkflowConditionType | CronWorkflowConditionType): JSX.Element {
    if (condition as WorkflowConditionType) {
        if (WarningWorkflowConditions.includes(condition as WorkflowConditionType)) {
            return <span className={'fa fa-exclamation-triangle'} style={{color: '#d7b700'}} />;
        }
        if (ErrorWorkflowConditions.includes(condition as WorkflowConditionType)) {
            return <span className={'fa fa-exclamation-circle'} style={{color: '#d70022'}} />;
        }
    }
    if (condition as CronWorkflowConditionType) {
        if (ErrorCronWorkflowConditions.includes(condition as CronWorkflowConditionType)) {
            return <span className={'fa fa-exclamation-circle'} style={{color: '#d70022'}} />;
        }
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

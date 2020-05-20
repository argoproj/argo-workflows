import * as React from 'react';
import {Condition, ConditionType} from '../../models';

interface Props {
    conditions: Condition[];
}

const WarningConditions: ConditionType[] = ['SpecWarning'];
const ErrorConditions: ConditionType[] = ['MetricsError', 'SubmissionError'];

export function hasWarningConditionBadge(conditions: Condition[]): boolean {
    if (conditions.length === 0) {
        return false;
    }

    for (const condition of conditions) {
        if (WarningConditions.includes(condition.type)) {
            return true;
        }
        if (ErrorConditions.includes(condition.type)) {
            return true;
        }
    }

    return false;
}

function getConditionIcon(condition: ConditionType): JSX.Element {
    if (WarningConditions.includes(condition as ConditionType)) {
        return <span className={'fa fa-exclamation-triangle'} style={{color: '#d7b700'}} />;
    }
    if (ErrorConditions.includes(condition as ConditionType)) {
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

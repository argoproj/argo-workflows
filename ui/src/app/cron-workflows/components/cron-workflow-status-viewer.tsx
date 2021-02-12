import * as kubernetes from 'argo-ui/src/models/kubernetes';
import * as React from 'react';
import {CronWorkflowSpec, CronWorkflowStatus} from '../../../models';
import {Timestamp} from '../../shared/components/timestamp';
import {ConditionsPanel} from '../../shared/conditions-panel';
import {WorkflowLink} from '../../workflows/components/workflow-link';
import {PrettySchedule} from './pretty-schedule';

const parser = require('cron-parser');
export const CronWorkflowStatusViewer = ({spec, status}: {spec: CronWorkflowSpec; status: CronWorkflowStatus}) => {
    if (status === null) {
        return null;
    }
    return (
        <div className='white-box'>
            <div className='white-box__details'>
                {[
                    {title: 'Active', value: status.active ? getCronWorkflowActiveWorkflowList(status.active) : <i>No Workflows Active</i>},
                    {
                        title: 'Schedule',
                        value: (
                            <>
                                <code>{spec.schedule}</code> <PrettySchedule schedule={spec.schedule} />
                            </>
                        )
                    },
                    {title: 'Last Scheduled Time', value: <Timestamp date={status.lastScheduledTime} />},
                    {
                        title: 'Next Scheduled Time',
                        value: (
                            <>
                                <Timestamp date={getNextScheduledTime(spec.schedule, spec.timezone)} /> (assumes workflow-controller is in UTC)
                            </>
                        )
                    },
                    {title: 'Conditions', value: <ConditionsPanel conditions={status.conditions} />}
                ].map(attr => (
                    <div className='row white-box__details-row' key={attr.title}>
                        <div className='columns small-3'>{attr.title}</div>
                        <div className='columns small-9'>{attr.value}</div>
                    </div>
                ))}
            </div>
        </div>
    );
};

function getCronWorkflowActiveWorkflowList(active: kubernetes.ObjectReference[]) {
    return active.reverse().map(activeWf => <WorkflowLink key={activeWf.uid} namespace={activeWf.namespace} name={activeWf.name} />);
}

function getNextScheduledTime(schedule: string, tz: string): string {
    let out = '';
    try {
        out = parser
            .parseExpression(schedule, {utc: !tz, tz})
            .next()
            .toISOString();
    } catch (e) {
        // Do nothing
    }
    return out;
}

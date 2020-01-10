import * as classNames from 'classnames';
import * as React from 'react';

import * as models from '../../../../models';

require('./workflow-steps.scss');

export interface WorkflowStepsProps {
    workflow: models.Workflow;
}

export const WorkflowSteps = (props: WorkflowStepsProps) => {
    const entryPointTemplate = props.workflow.spec.templates.find(template => template.name === props.workflow.spec.entrypoint) || {steps: [] as models.WorkflowStep[][]};
    const phase = props.workflow.status.phase;
    let isSucceeded = false;
    let isFailed = false;
    let isRunning = false;
    if (phase === models.NODE_PHASE.RUNNING) {
        isRunning = true;
    } else {
        isSucceeded = phase === models.NODE_PHASE.SUCCEEDED;
        isFailed = !isSucceeded;
    }
    const steps = (entryPointTemplate.steps || [])
        .map(group => group[0])
        .map(step => ({name: step.name, isSucceeded, isFailed, isRunning}))
        .slice(0, 3);

    return (
        <div className='workflow-steps'>
            <div className='workflow-steps__title'>
                <div className='workflow-steps__icon'>
                    <i className='ax-icon-job' aria-hidden='true' />
                </div>
                <div className='workflow-steps__description'>
                    <div className='workflow-steps__description-title'>{props.workflow.metadata.name}</div>
                </div>
            </div>
            <div className='workflow-steps__timeline'>
                <div className='workflow-steps__step-dots'>
                    <div className='workflow-steps__step-circle workflow-steps__step-circle-small' />
                    <div className='workflow-steps__step-circle workflow-steps__step-circle-small' />
                    <div className='workflow-steps__step-circle workflow-steps__step-circle-small' />
                    <div className='workflow-steps__step-name'>&nbsp;</div>
                </div>
                {steps.map(step => (
                    <div
                        key={step.name}
                        className={classNames('workflow-steps__step', {
                            'workflow-steps__step--succeeded': step.isSucceeded,
                            'workflow-steps__step--failed': step.isFailed,
                            'workflow-steps__step--running': step.isRunning
                        })}>
                        <div
                            className={classNames('workflow-steps__step-circle', {
                                'workflow-steps__step-circle--succeeded': step.isSucceeded,
                                'workflow-steps__step-circle--failed': step.isFailed,
                                'workflow-steps__step-circle--running': step.isRunning
                            })}
                        />
                        <div className='workflow-steps__step-name'>{step.name}</div>
                    </div>
                ))}
            </div>
        </div>
    );
};

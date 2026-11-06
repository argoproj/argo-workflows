import {render, screen} from '@testing-library/react';
import * as React from 'react';
import {MemoryRouter} from 'react-router-dom';

import {Condition, CronWorkflow} from '../shared/models';
import {CronWorkflowRow} from './cron-workflow-row';

function workflow(conditions?: Condition[], suspend = false, schedules = ['0 * * * *'], resolvedSchedules?: {[schedule: string]: string}): CronWorkflow {
    return {
        metadata: {
            name: 'my-cron-workflow',
            namespace: 'default'
        },
        spec: {
            schedules,
            suspend,
            workflowSpec: {}
        },
        status:
            conditions || resolvedSchedules
                ? {
                      active: [],
                      conditions,
                      lastScheduledTime: null,
                      resolvedSchedules
                  }
                : undefined
    };
}

function renderRow(cronWorkflow: CronWorkflow) {
    return render(
        <MemoryRouter>
            <CronWorkflowRow workflow={cronWorkflow} displayISOFormatCreation={false} displayISOFormatNextScheduled={false} />
        </MemoryRouter>
    );
}

describe('CronWorkflowRow', () => {
    it('shows an error icon and condition message for an error condition', () => {
        const {container} = renderRow(
            workflow([
                {
                    type: 'SpecError',
                    status: 'True',
                    message: "template name 'missing' undefined"
                }
            ])
        );

        const error = screen.getByTitle("template name 'missing' undefined");
        expect(error.querySelector('.status-icon--error')).toBeInTheDocument();
        expect(container.querySelector('.fa-clock')).not.toBeInTheDocument();
    });

    it('shows the clock icon when there are no error conditions', () => {
        const {container} = renderRow(workflow());

        expect(container.querySelector('.fa-clock')).toBeInTheDocument();
        expect(container.querySelector('.status-icon--error')).not.toBeInTheDocument();
    });

    it('describes the schedule a hashed schedule resolved to', () => {
        renderRow(workflow(undefined, false, ['H H * * *'], {'H H * * *': '9 6 * * *'}));

        // the schedule is shown as configured, but described as it runs
        expect(screen.getByText('H H * * *')).toBeInTheDocument();
        expect(screen.getByTitle('At 06:09 AM')).toBeInTheDocument();
    });

    it('marks a hashed schedule which is not resolved yet', () => {
        renderRow(workflow(undefined, false, ['H H * * *']));

        expect(screen.getByText('hashed schedule')).toBeInTheDocument();
    });

    it('ignores the resolution of a schedule which has since been edited', () => {
        renderRow(workflow(undefined, false, ['H H * * *'], {'0 0 * * *': '9 6 * * *'}));

        expect(screen.getByText('hashed schedule')).toBeInTheDocument();
    });

    it('keeps the pause icon for a suspended workflow without errors', () => {
        const {container} = renderRow(workflow(undefined, true));

        expect(container.querySelector('.fa-pause')).toBeInTheDocument();
        expect(container.querySelector('.status-icon--error')).not.toBeInTheDocument();
    });
});

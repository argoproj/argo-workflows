import {fireEvent, render, screen} from '@testing-library/react';
import React from 'react';

import {CronWorkflowSpec} from '../shared/models';
import {CronWorkflowSpecEditor} from './cron-workflow-spec-editor';

const baseSpec: CronWorkflowSpec = {
    schedules: ['* * * * *'],
    workflowSpec: {
        entrypoint: 'main',
        templates: [{name: 'main', container: {name: 'main', image: 'argoproj/argosay:v2', command: ['cowsay']}}]
    }
};

describe('CronWorkflowSpecEditor', () => {
    it('does not display NaN when startingDeadlineSeconds is undefined', () => {
        render(<CronWorkflowSpecEditor spec={baseSpec} onChange={() => undefined} />);
        const inputs = screen.getAllByRole('textbox');
        inputs.forEach(input => {
            expect(input).not.toHaveValue('NaN');
        });
    });

    it('does not display NaN when numeric fields are explicitly undefined', () => {
        const spec: CronWorkflowSpec = {
            ...baseSpec,
            startingDeadlineSeconds: undefined,
            successfulJobsHistoryLimit: undefined,
            failedJobsHistoryLimit: undefined
        };
        render(<CronWorkflowSpecEditor spec={spec} onChange={() => undefined} />);
        const inputs = screen.getAllByRole('textbox');
        inputs.forEach(input => {
            expect(input).not.toHaveValue('NaN');
        });
    });

    it('displays numeric values correctly when set', () => {
        const spec: CronWorkflowSpec = {
            ...baseSpec,
            startingDeadlineSeconds: 60,
            successfulJobsHistoryLimit: 3,
            failedJobsHistoryLimit: 1
        };
        render(<CronWorkflowSpecEditor spec={spec} onChange={() => undefined} />);
        expect(screen.getByDisplayValue('60')).toBeInTheDocument();
        expect(screen.getByDisplayValue('3')).toBeInTheDocument();
        expect(screen.getByDisplayValue('1')).toBeInTheDocument();
    });

    it('calls onChange with undefined when a numeric field is cleared', () => {
        const handleChange = jest.fn();
        const spec: CronWorkflowSpec = {
            ...baseSpec,
            startingDeadlineSeconds: 60
        };
        render(<CronWorkflowSpecEditor spec={spec} onChange={handleChange} />);
        const input = screen.getByDisplayValue('60');
        fireEvent.change(input, {target: {value: ''}});
        expect(handleChange).toHaveBeenCalledWith(
            expect.objectContaining({startingDeadlineSeconds: undefined})
        );
    });

    it('calls onChange with parsed number when a valid value is entered', () => {
        const handleChange = jest.fn();
        const spec: CronWorkflowSpec = {
            ...baseSpec,
            successfulJobsHistoryLimit: 3
        };
        render(<CronWorkflowSpecEditor spec={spec} onChange={handleChange} />);
        const input = screen.getByDisplayValue('3');
        fireEvent.change(input, {target: {value: '5'}});
        expect(handleChange).toHaveBeenCalledWith(
            expect.objectContaining({successfulJobsHistoryLimit: 5})
        );
    });
});

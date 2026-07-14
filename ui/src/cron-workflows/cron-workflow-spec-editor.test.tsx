import {fireEvent, render, screen} from '@testing-library/react';
import {Autocomplete} from 'argo-ui/src/components/autocomplete/autocomplete';
import React from 'react';

import {CronWorkflowSpec} from '../shared/models';
import {CronWorkflowSpecEditor} from './cron-workflow-spec-editor';

jest.mock('argo-ui/src/components/autocomplete/autocomplete', () => ({
    Autocomplete: jest.fn((props: React.InputHTMLAttributes<HTMLInputElement> & {onSelect: (value: string) => void}) => (
        <input data-testid='timezone-autocomplete' value={props.value as string} onChange={props.onChange} />
    ))
}));

const baseSpec: CronWorkflowSpec = {
    schedules: ['* * * * *'],
    workflowSpec: {
        entrypoint: 'main',
        templates: [{name: 'main', container: {name: 'main', image: 'argoproj/argosay:v2', command: ['cowsay']}}]
    }
};

describe('CronWorkflowSpecEditor', () => {
    beforeEach(() => {
        (Autocomplete as unknown as jest.Mock).mockClear();
    });

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
        expect(handleChange).toHaveBeenCalledWith(expect.objectContaining({startingDeadlineSeconds: undefined}));
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
        expect(handleChange).toHaveBeenCalledWith(expect.objectContaining({successfulJobsHistoryLimit: 5}));
    });

    it('filters timezone autocomplete suggestions by the entered prefix', () => {
        render(<CronWorkflowSpecEditor spec={{...baseSpec, timezone: 'Asia/T'}} onChange={() => undefined} />);

        const props = (Autocomplete as unknown as jest.Mock).mock.calls[0][0];
        expect(props.items).toContain('Asia/Tokyo');
        expect(props.items.every((timezone: string) => timezone.startsWith('Asia/T'))).toBe(true);
    });

    it('updates the timezone from typed input and autocomplete selection', () => {
        const handleChange = jest.fn();
        render(<CronWorkflowSpecEditor spec={baseSpec} onChange={handleChange} />);

        fireEvent.change(screen.getByTestId('timezone-autocomplete'), {target: {value: 'Asia/T'}});
        expect(handleChange).toHaveBeenCalledWith(expect.objectContaining({timezone: 'Asia/T'}));

        const props = (Autocomplete as unknown as jest.Mock).mock.calls[0][0];
        props.onSelect('Asia/Tokyo');
        expect(handleChange).toHaveBeenCalledWith(expect.objectContaining({timezone: 'Asia/Tokyo'}));
    });
});

import {fireEvent, render, screen} from '@testing-library/react';
import * as React from 'react';

import {CronWorkflowDeleteConfirmation} from './cron-workflow-delete-confirmation';

describe('CronWorkflowDeleteConfirmation', () => {
    it('does not keep created Workflows by default', () => {
        render(<CronWorkflowDeleteConfirmation onKeepWorkflowsChange={jest.fn()} />);

        expect(screen.getByText('Deleting it also deletes all non-archived Workflows it created.')).toBeInTheDocument();
        expect(screen.getByLabelText('Keep Workflows created by this CronWorkflow')).not.toBeChecked();
    });

    it('reports changes to the keep Workflows option', () => {
        const onKeepWorkflowsChange = jest.fn();
        render(<CronWorkflowDeleteConfirmation onKeepWorkflowsChange={onKeepWorkflowsChange} />);

        const checkbox = screen.getByLabelText('Keep Workflows created by this CronWorkflow');
        fireEvent.click(checkbox);

        expect(checkbox).toBeChecked();
        expect(onKeepWorkflowsChange).toHaveBeenCalledWith(true);

        fireEvent.click(checkbox);

        expect(checkbox).not.toBeChecked();
        expect(onKeepWorkflowsChange).toHaveBeenLastCalledWith(false);
    });
});

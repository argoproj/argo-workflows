import {fireEvent, render, screen} from '@testing-library/react';
import React from 'react';

import {WorkflowPhase} from '../../../shared/models';
import {WorkflowFilters} from './workflow-filters';

jest.mock('../../../shared/services', () => ({
    services: {
        workflowTemplate: {list: jest.fn().mockResolvedValue({items: []})},
        cronWorkflows: {list: jest.fn().mockResolvedValue([])}
    }
}));

function renderFilters(overrides: Partial<React.ComponentProps<typeof WorkflowFilters>> = {}) {
    const props: React.ComponentProps<typeof WorkflowFilters> = {
        workflows: [],
        namespace: 'argo',
        phaseItems: [] as WorkflowPhase[],
        phases: [] as WorkflowPhase[],
        labels: [],
        createdAfter: undefined,
        finishedBefore: undefined,
        setNamespace: jest.fn(),
        setPhases: jest.fn(),
        setLabels: jest.fn(),
        setCreatedAfter: jest.fn(),
        setFinishedBefore: jest.fn(),
        nameFilter: 'Contains',
        nameValue: '',
        setNameFilter: jest.fn(),
        setNameValue: jest.fn(),
        ...overrides
    };
    return {props, ...render(<WorkflowFilters {...props} />)};
}

describe('WorkflowFilters date pickers', () => {
    it('mounts both react-datepicker inputs without throwing', () => {
        renderFilters();
        expect(screen.getByPlaceholderText('From')).toBeInTheDocument();
        expect(screen.getByPlaceholderText('To')).toBeInTheDocument();
    });

    it('passes a native Date to setCreatedAfter when a date is typed', () => {
        const {props} = renderFilters();
        const input = screen.getByPlaceholderText('From');
        fireEvent.change(input, {target: {value: '15 Jan 2024'}});
        expect(props.setCreatedAfter).toHaveBeenCalled();
        const arg = (props.setCreatedAfter as jest.Mock).mock.calls[0][0];
        expect(arg).toBeInstanceOf(Date);
    });

    it('shows the selected date in the input', () => {
        renderFilters({createdAfter: new Date(2024, 0, 15)});
        expect(screen.getByDisplayValue('15 Jan 2024')).toBeInTheDocument();
    });
});

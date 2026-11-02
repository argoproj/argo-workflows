import {fireEvent, render, screen} from '@testing-library/react';
import React from 'react';

import {FilterDropDown, FilterDropSection} from './filter-drop-down';

describe('FilterDropDown', () => {
    beforeEach(() => {
        Object.defineProperty(HTMLElement.prototype, 'offsetHeight', {configurable: true, value: 50});
        Object.defineProperty(HTMLElement.prototype, 'offsetWidth', {configurable: true, value: 100});
        Object.defineProperty(HTMLElement.prototype, 'getBoundingClientRect', {
            configurable: true,
            value: () => ({top: 0, left: 0, bottom: 50, right: 100, width: 100, height: 50})
        });
    });

    const sections: FilterDropSection[] = [
        {
            title: 'Status',
            values: {Running: true, Failed: false},
            onChange: jest.fn()
        }
    ];

    it('renders the filter icon', () => {
        render(<FilterDropDown sections={sections} />);
        expect(document.querySelector('.argo-icon-filter')).toBeInTheDocument();
    });

    it('renders the chevron icon', () => {
        render(<FilterDropDown sections={sections} />);
        expect(document.querySelector('.fa-angle-down')).toBeInTheDocument();
    });

    it('opens dropdown and shows section title on click', () => {
        render(<FilterDropDown sections={sections} />);
        const anchor = document.querySelector('.argo-dropdown__anchor');
        fireEvent.click(anchor);
        expect(screen.getByText('Status')).toBeInTheDocument();
    });

    it('shows checkbox labels for filter values', () => {
        render(<FilterDropDown sections={sections} />);
        const anchor = document.querySelector('.argo-dropdown__anchor');
        fireEvent.click(anchor);
        expect(screen.getByText('Failed')).toBeInTheDocument();
        expect(screen.getByText('Running')).toBeInTheDocument();
    });

    it('calls onChange when a checkbox is toggled', () => {
        const onChange = jest.fn();
        const testSections: FilterDropSection[] = [{title: 'Phase', values: {Pending: false}, onChange}];
        render(<FilterDropDown sections={testSections} />);

        const anchor = document.querySelector('.argo-dropdown__anchor');
        fireEvent.click(anchor);

        const checkbox = screen.getByLabelText('Pending');
        fireEvent.click(checkbox);
        expect(onChange).toHaveBeenCalledWith('Pending', true);
    });
});

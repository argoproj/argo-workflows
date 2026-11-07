import {fireEvent, render, screen} from '@testing-library/react';
import React from 'react';

import {ToolbarActions} from './toolbar-actions';

describe('ToolbarActions', () => {
    it('renders actions as buttons and links as anchors, in order', () => {
        const action = jest.fn();
        render(
            <ToolbarActions
                items={[
                    {title: 'Logs', iconClassName: 'fa fa-bars', action},
                    {title: 'Grafana', iconClassName: 'fa fa-external-link-alt', href: 'https://grafana/my-wf', target: '_blank'},
                    {title: 'Delete', action: jest.fn(), disabled: true}
                ]}>
                <span>tools</span>
            </ToolbarActions>
        );

        const button = screen.getByRole('button', {name: 'Logs'});
        fireEvent.click(button);
        expect(action).toHaveBeenCalledTimes(1);

        const link = screen.getByRole('link', {name: 'Grafana'});
        expect(link.tagName).toBe('A');
        expect(link).toHaveAttribute('href', 'https://grafana/my-wf');
        expect(link).toHaveAttribute('target', '_blank');
        expect(link).toHaveClass('argo-button', 'argo-button--base');

        expect(screen.getByRole('button', {name: 'Delete'})).toBeDisabled();
        expect(button.compareDocumentPosition(link) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
        expect(screen.getByText('tools')).toBeInTheDocument();
    });
});

import {fireEvent, render, screen} from '@testing-library/react';
import React from 'react';

import {Button} from './button';

describe('Button', () => {
    it('renders children and responds to click', () => {
        const handleClick = jest.fn();
        render(<Button onClick={handleClick}>Click me</Button>);
        const button = screen.getByText('Click me');
        expect(button).toBeInTheDocument();
        fireEvent.click(button);
        expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it('renders a button element without an href', () => {
        render(<Button onClick={jest.fn()}>Click me</Button>);
        expect(screen.getByRole('button', {name: 'Click me'})).toBeInTheDocument();
        expect(screen.queryByRole('link')).not.toBeInTheDocument();
    });

    it('renders an anchor element with an href', () => {
        render(
            <Button href='https://example.com/logs' target='_blank'>
                Logs
            </Button>
        );
        const link = screen.getByRole('link', {name: 'Logs'});
        expect(link.tagName).toBe('A');
        expect(link).toHaveAttribute('href', 'https://example.com/logs');
        expect(link).toHaveAttribute('target', '_blank');
        expect(link).toHaveClass('argo-button', 'argo-button--base');
        expect(screen.queryByRole('button')).not.toBeInTheDocument();
    });
});

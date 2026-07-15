import {render} from '@testing-library/react';
import React from 'react';

import {BigButton} from './big-button';

describe('BigButton', () => {
    it('renders with correct title, attributes, and icon, and responds to click', () => {
        const handleClick = jest.fn();
        const href = 'https://example.com';
        const {getByRole} = render(<BigButton icon='check' title='Example' onClick={handleClick} href={href} />);

        // Find the link by its accessible name (the title text)
        const link = getByRole('link', {name: 'Example'});
        expect(link).toBeInTheDocument();
        expect(link).toHaveAttribute('href', href);
        expect(link).toHaveAttribute('target', '_blank');
        expect(link).toHaveAttribute('rel', 'noreferrer');

        // Icon element should be inside the link with correct classes
        const icon = link.querySelector('i');
        expect(icon).toBeInTheDocument();
        expect(icon).toHaveClass('fa', 'fa-check');

        // Click should trigger onClick handler
        link.click();
        expect(handleClick).toHaveBeenCalledTimes(1);
    });
});

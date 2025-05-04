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
});

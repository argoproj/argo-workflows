import {fireEvent, render, screen} from '@testing-library/react';
import React from 'react';

import {ThemeToggle} from './theme-toggle';

describe('ThemeToggle', () => {
    beforeEach(() => localStorage.clear());

    it('is a labelled button that cycles system, light, dark', () => {
        render(<ThemeToggle />);
        const button = screen.getByRole('button', {name: 'Theme: system'});
        fireEvent.click(button);
        fireEvent.click(screen.getByRole('button', {name: 'Theme: light'}));
        expect(screen.getByRole('button', {name: 'Theme: dark'})).toBeInTheDocument();
        fireEvent.click(screen.getByRole('button', {name: 'Theme: dark'}));
        expect(screen.getByRole('button', {name: 'Theme: system'})).toBeInTheDocument();
    });
});

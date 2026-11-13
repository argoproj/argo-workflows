import * as React from 'react';
import type {ReactNode} from 'react';

function load(stored?: string) {
    localStorage.clear();
    if (stored !== undefined) {
        localStorage.setItem('theme', stored);
    }
    let rtl: typeof import('@testing-library/react');
    let argo: typeof import('argo-ui/v2/shared/context/theme');
    let toggle: typeof import('./theme-toggle');
    jest.isolateModules(() => {
        rtl = require('@testing-library/react');
        argo = require('argo-ui/v2/shared/context/theme');
        toggle = require('./theme-toggle');
    });
    const {ThemeProvider} = argo!;
    const {ThemeToggle} = toggle!;
    const wrapper = ({children}: {children?: ReactNode}) => <ThemeProvider>{children}</ThemeProvider>;
    return {...rtl!, ui: <ThemeToggle />, wrapper};
}

describe('ThemeToggle', () => {
    it('is a labelled button that offers dark mode while the theme is light', () => {
        const {render, ui, wrapper} = load();
        const view = render(ui, {wrapper});
        const button = view.getByRole('button', {name: 'Switch to dark theme'});
        expect(button.querySelector('.fa-moon')).not.toBeNull();
        view.unmount();
    });

    it('offers light mode while the theme is dark', () => {
        const {render, ui, wrapper} = load('"dark"');
        const view = render(ui, {wrapper});
        const button = view.getByRole('button', {name: 'Switch to light theme'});
        expect(button.querySelector('.fa-sun')).not.toBeNull();
        view.unmount();
    });

    it('switches the theme and saves it when clicked', () => {
        const {render, fireEvent, ui, wrapper} = load();
        const view = render(ui, {wrapper});
        fireEvent.click(view.getByRole('button', {name: 'Switch to dark theme'}));
        expect(JSON.parse(localStorage.getItem('theme')!)).toBe('dark');
        fireEvent.click(view.getByRole('button', {name: 'Switch to light theme'}));
        expect(JSON.parse(localStorage.getItem('theme')!)).toBe('light');
        view.unmount();
    });
});

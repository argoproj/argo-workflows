import * as React from 'react';

import {useThemeContext} from '../theme-context';
import {ThemeValue} from '../use-theme';

import './theme-selector.scss';

interface ThemeOption {
    value: ThemeValue;
    label: string;
    icon: string;
}

const themeOptions: ThemeOption[] = [
    {value: 'light', label: 'Light', icon: 'fa-sun'},
    {value: 'dark', label: 'Dark', icon: 'fa-moon'},
    {value: 'auto', label: 'System', icon: 'fa-desktop'}
];

export function ThemeSelector() {
    const {theme, setTheme} = useThemeContext();

    return (
        <div className='theme-selector'>
            {themeOptions.map(option => (
                <button
                    key={option.value}
                    className={`argo-button argo-button--base-o theme-selector__button ${theme === option.value ? 'theme-selector__button--selected' : ''}`}
                    onClick={() => setTheme(option.value)}
                    title={option.label}>
                    <i className={`fa ${option.icon}`} /> {option.label}
                </button>
            ))}
        </div>
    );
}

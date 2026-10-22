import * as React from 'react';

import {setThemePreference, ThemePreference, useTheme} from '../theme';

const next: {[key in ThemePreference]: ThemePreference} = {system: 'light', light: 'dark', dark: 'system'};
const icons: {[key in ThemePreference]: string} = {system: 'fa-desktop', light: 'fa-sun', dark: 'fa-moon'};

export function ThemeToggle() {
    const {preference} = useTheme();
    return (
        <button type='button' className='theme-toggle' title={`Theme: ${preference}`} aria-label={`Theme: ${preference}`} onClick={() => setThemePreference(next[preference])}>
            <i className={`fa ${icons[preference]}`} />
        </button>
    );
}

import {Theme, ThemeContext} from 'argo-ui/v2/shared/context/theme';
import * as React from 'react';

export function ThemeToggle() {
    const {theme, set} = React.useContext(ThemeContext);
    const dark = theme === Theme.Dark;
    const label = dark ? 'Switch to light theme' : 'Switch to dark theme';
    return (
        <button type='button' className='theme-toggle' title={label} aria-label={label} onClick={() => set(dark ? Theme.Light : Theme.Dark)}>
            <i className={`fa ${dark ? 'fa-sun' : 'fa-moon'}`} />
        </button>
    );
}

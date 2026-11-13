import {Theme, THEME_KEY, ThemeContext} from 'argo-ui/v2/shared/context/theme';
import {useContext, useEffect, useRef} from 'react';

function hasStoredTheme() {
    try {
        return localStorage.getItem(THEME_KEY) !== null;
    } catch {
        return false;
    }
}

// Defaults argo-ui's theme to the OS setting, and follows it until the user picks a theme.
export function useThemeSync() {
    const {theme, set} = useContext(ThemeContext);
    const setRef = useRef(set);
    setRef.current = set;
    const following = useRef<boolean>();
    if (following.current === undefined) {
        following.current = !hasStoredTheme();
    }
    const applied = useRef<Theme>();

    useEffect(() => {
        if (applied.current !== undefined && theme !== applied.current) {
            following.current = false;
        }
    }, [theme]);

    useEffect(() => {
        const media = window.matchMedia?.('(prefers-color-scheme: dark)');
        if (!media || !following.current) {
            return;
        }
        const apply = () => {
            if (following.current) {
                applied.current = media.matches ? Theme.Dark : Theme.Light;
                setRef.current(applied.current);
            }
        };
        apply();
        media.addEventListener('change', apply);
        return () => media.removeEventListener('change', apply);
    }, []);

    useEffect(() => {
        document.body.classList.toggle('theme-dark', theme === Theme.Dark);
    }, [theme]);

    return theme;
}

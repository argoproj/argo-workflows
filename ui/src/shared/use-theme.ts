import {useCallback, useEffect, useState} from 'react';

export type ThemeValue = 'light' | 'dark' | 'auto';

const STORAGE_KEY = 'argo-workflows-theme';

function getSystemTheme(): 'light' | 'dark' {
    if (typeof window !== 'undefined' && window.matchMedia) {
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
    }
    return 'light';
}

function getStoredTheme(): ThemeValue {
    if (typeof window !== 'undefined') {
        const stored = window.localStorage.getItem(STORAGE_KEY);
        if (stored === 'light' || stored === 'dark' || stored === 'auto') {
            return stored;
        }
    }
    return 'auto';
}

export interface UseThemeResult {
    theme: ThemeValue;
    resolvedTheme: 'light' | 'dark';
    setTheme: (theme: ThemeValue) => void;
}

export function useTheme(): UseThemeResult {
    const [theme, setThemeState] = useState<ThemeValue>(getStoredTheme);
    const [systemTheme, setSystemTheme] = useState<'light' | 'dark'>(getSystemTheme);

    const setTheme = useCallback((newTheme: ThemeValue) => {
        setThemeState(newTheme);
        if (typeof window !== 'undefined') {
            window.localStorage.setItem(STORAGE_KEY, newTheme);
        }
    }, []);

    useEffect(() => {
        if (typeof window === 'undefined' || !window.matchMedia) {
            return;
        }

        const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
        const handleChange = (e: MediaQueryListEvent) => {
            setSystemTheme(e.matches ? 'dark' : 'light');
        };

        mediaQuery.addEventListener('change', handleChange);
        return () => mediaQuery.removeEventListener('change', handleChange);
    }, []);

    const resolvedTheme = theme === 'auto' ? systemTheme : theme;

    return {theme, resolvedTheme, setTheme};
}

import {createContext, useContext} from 'react';

import {ThemeValue} from './use-theme';

export interface ThemeContextValue {
    theme: ThemeValue;
    resolvedTheme: 'light' | 'dark';
    setTheme: (theme: ThemeValue) => void;
}

export const ThemeContext = createContext<ThemeContextValue | null>(null);

export function useThemeContext(): ThemeContextValue {
    const context = useContext(ThemeContext);
    if (!context) {
        throw new Error('useThemeContext must be used within a ThemeContext.Provider');
    }
    return context;
}

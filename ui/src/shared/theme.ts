import {THEME_KEY} from 'argo-ui/v2/shared/context/theme';
import {useEffect, useState} from 'react';

export type ThemePreference = 'system' | 'light' | 'dark';

const media = window.matchMedia?.('(prefers-color-scheme: dark)');
const listeners = new Set<() => void>();

function readPreference(): ThemePreference {
    try {
        const stored = localStorage.getItem(THEME_KEY);
        const value = stored === null ? null : JSON.parse(stored);
        return value === 'light' || value === 'dark' ? value : 'system';
    } catch {
        return 'system';
    }
}

let preference = readPreference();

function notify() {
    listeners.forEach(listener => listener());
}

media?.addEventListener('change', notify);

function store(value: ThemePreference) {
    try {
        localStorage.setItem(THEME_KEY, JSON.stringify(value));
        return true;
    } catch {
        return false;
    }
}

export function setThemePreference(value: ThemePreference) {
    preference = value;
    store(value);
    notify();
}

function current() {
    const theme: 'light' | 'dark' = preference === 'system' ? (media?.matches ? 'dark' : 'light') : preference;
    return {preference, theme};
}

export function useTheme() {
    const [state, setState] = useState(current);
    useEffect(() => {
        const listener = () => setState(current());
        listeners.add(listener);
        return () => {
            listeners.delete(listener);
        };
    }, []);
    return state;
}

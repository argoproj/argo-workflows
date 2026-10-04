import {useEffect, useState} from 'react';

export type ThemePreference = 'system' | 'light' | 'dark';

const storageKey = 'theme';
const media = window.matchMedia?.('(prefers-color-scheme: dark)');
const listeners = new Set<() => void>();

let preference: ThemePreference = (localStorage.getItem(storageKey) as ThemePreference) || 'system';

function notify() {
    listeners.forEach(listener => listener());
}

media?.addEventListener('change', notify);

export function setThemePreference(value: ThemePreference) {
    preference = value;
    localStorage.setItem(storageKey, value);
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

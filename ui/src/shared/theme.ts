import {useEffect, useState} from 'react';

export type ThemePreference = 'system' | 'light' | 'dark';

const storageKey = 'theme';
const media = window.matchMedia?.('(prefers-color-scheme: dark)');
const listeners = new Set<() => void>();

function readPreference(): ThemePreference {
    try {
        const stored = localStorage.getItem(storageKey);
        return stored === 'light' || stored === 'dark' ? stored : 'system';
    } catch {
        return 'system';
    }
}

let preference = readPreference();

function notify() {
    listeners.forEach(listener => listener());
}

media?.addEventListener('change', notify);

export function setThemePreference(value: ThemePreference) {
    preference = value;
    try {
        localStorage.setItem(storageKey, value);
    } catch {
        // storage is unavailable, so the preference only lasts for this session
    }
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

import {act, renderHook} from '@testing-library/react';

type Theme = typeof import('./theme');

function loadTheme(systemDark: boolean) {
    let change: () => void = () => {};
    const media = {
        matches: systemDark,
        addEventListener: (_: string, listener: () => void) => {
            change = listener;
        }
    };
    window.matchMedia = jest.fn().mockReturnValue(media);
    let theme: Theme;
    jest.isolateModules(() => {
        theme = require('./theme');
    });
    return {
        theme: theme!,
        setSystemDark: (value: boolean) => {
            media.matches = value;
            change();
        }
    };
}

describe('theme', () => {
    beforeEach(() => localStorage.clear());

    it('defaults to the system preference', () => {
        const {theme} = loadTheme(true);
        const {result} = renderHook(() => theme.useTheme());
        expect(result.current).toEqual({preference: 'system', theme: 'dark'});
    });

    it('follows the system when it changes', () => {
        const {theme, setSystemDark} = loadTheme(false);
        const {result} = renderHook(() => theme.useTheme());
        expect(result.current.theme).toBe('light');
        act(() => setSystemDark(true));
        expect(result.current.theme).toBe('dark');
    });

    it('stores an explicit preference and ignores the system', () => {
        const {theme, setSystemDark} = loadTheme(false);
        const {result} = renderHook(() => theme.useTheme());
        act(() => theme.setThemePreference('dark'));
        expect(result.current).toEqual({preference: 'dark', theme: 'dark'});
        expect(localStorage.getItem('theme')).toBe('dark');
        act(() => setSystemDark(false));
        expect(result.current.theme).toBe('dark');
    });

    it('restores the stored preference', () => {
        localStorage.setItem('theme', 'light');
        const {theme} = loadTheme(true);
        const {result} = renderHook(() => theme.useTheme());
        expect(result.current).toEqual({preference: 'light', theme: 'light'});
    });

    it('ignores an invalid stored preference', () => {
        localStorage.setItem('theme', 'purple');
        const {theme} = loadTheme(false);
        const {result} = renderHook(() => theme.useTheme());
        expect(result.current).toEqual({preference: 'system', theme: 'light'});
    });

    it('keeps working when storage throws', () => {
        jest.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
            throw new Error('denied');
        });
        jest.spyOn(Storage.prototype, 'setItem').mockImplementation(() => {
            throw new Error('denied');
        });
        const {theme} = loadTheme(false);
        const {result} = renderHook(() => theme.useTheme());
        expect(result.current.preference).toBe('system');
        act(() => theme.setThemePreference('dark'));
        expect(result.current).toEqual({preference: 'dark', theme: 'dark'});
        jest.restoreAllMocks();
    });
});

import {act, renderHook} from '@testing-library/react';

import {ThemeValue, useTheme} from './use-theme';

describe('useTheme', () => {
    const STORAGE_KEY = 'argo-workflows-theme';
    let mockMatchMedia: jest.Mock;
    let mediaQueryListeners: Array<(e: MediaQueryListEvent) => void>;

    beforeEach(() => {
        localStorage.clear();
        mediaQueryListeners = [];

        mockMatchMedia = jest.fn().mockImplementation((query: string) => ({
            matches: query === '(prefers-color-scheme: dark)' ? false : false,
            media: query,
            addEventListener: (_event: string, listener: (e: MediaQueryListEvent) => void) => {
                mediaQueryListeners.push(listener);
            },
            removeEventListener: (_event: string, listener: (e: MediaQueryListEvent) => void) => {
                mediaQueryListeners = mediaQueryListeners.filter(l => l !== listener);
            }
        }));

        Object.defineProperty(window, 'matchMedia', {
            writable: true,
            value: mockMatchMedia
        });
    });

    afterEach(() => {
        jest.restoreAllMocks();
    });

    describe('initial state', () => {
        it('defaults to auto theme when no stored value', () => {
            const {result} = renderHook(() => useTheme());

            expect(result.current.theme).toBe('auto');
            expect(result.current.resolvedTheme).toBe('light');
        });

        it('reads stored theme from localStorage', () => {
            localStorage.setItem(STORAGE_KEY, 'dark');

            const {result} = renderHook(() => useTheme());

            expect(result.current.theme).toBe('dark');
            expect(result.current.resolvedTheme).toBe('dark');
        });

        it('ignores invalid stored values', () => {
            localStorage.setItem(STORAGE_KEY, 'invalid-theme');

            const {result} = renderHook(() => useTheme());

            expect(result.current.theme).toBe('auto');
        });
    });

    describe('setTheme', () => {
        it('updates theme state', () => {
            const {result} = renderHook(() => useTheme());

            act(() => {
                result.current.setTheme('dark');
            });

            expect(result.current.theme).toBe('dark');
            expect(result.current.resolvedTheme).toBe('dark');
        });

        it('persists theme to localStorage', () => {
            const {result} = renderHook(() => useTheme());

            act(() => {
                result.current.setTheme('light');
            });

            expect(localStorage.getItem(STORAGE_KEY)).toBe('light');
        });

        it.each(['light', 'dark', 'auto'] as ThemeValue[])('handles %s theme', theme => {
            const {result} = renderHook(() => useTheme());

            act(() => {
                result.current.setTheme(theme);
            });

            expect(result.current.theme).toBe(theme);
            expect(localStorage.getItem(STORAGE_KEY)).toBe(theme);
        });
    });

    describe('system theme detection', () => {
        it('detects dark system preference', () => {
            mockMatchMedia.mockImplementation((query: string) => ({
                matches: query === '(prefers-color-scheme: dark)',
                media: query,
                addEventListener: jest.fn(),
                removeEventListener: jest.fn()
            }));

            const {result} = renderHook(() => useTheme());

            expect(result.current.theme).toBe('auto');
            expect(result.current.resolvedTheme).toBe('dark');
        });

        it('detects light system preference', () => {
            mockMatchMedia.mockImplementation((query: string) => ({
                matches: false,
                media: query,
                addEventListener: jest.fn(),
                removeEventListener: jest.fn()
            }));

            const {result} = renderHook(() => useTheme());

            expect(result.current.theme).toBe('auto');
            expect(result.current.resolvedTheme).toBe('light');
        });

        it('responds to system theme changes', () => {
            const {result} = renderHook(() => useTheme());

            expect(result.current.resolvedTheme).toBe('light');

            act(() => {
                mediaQueryListeners.forEach(listener => {
                    listener({matches: true} as MediaQueryListEvent);
                });
            });

            expect(result.current.resolvedTheme).toBe('dark');
        });

        it('cleans up event listener on unmount', () => {
            const removeEventListener = jest.fn();
            mockMatchMedia.mockImplementation(() => ({
                matches: false,
                addEventListener: jest.fn(),
                removeEventListener
            }));

            const {unmount} = renderHook(() => useTheme());

            unmount();

            expect(removeEventListener).toHaveBeenCalled();
        });
    });

    describe('resolvedTheme', () => {
        it('returns explicit theme when not auto', () => {
            const {result} = renderHook(() => useTheme());

            act(() => {
                result.current.setTheme('dark');
            });

            expect(result.current.resolvedTheme).toBe('dark');

            act(() => {
                result.current.setTheme('light');
            });

            expect(result.current.resolvedTheme).toBe('light');
        });

        it('returns system theme when auto', () => {
            mockMatchMedia.mockImplementation((query: string) => ({
                matches: query === '(prefers-color-scheme: dark)',
                media: query,
                addEventListener: jest.fn(),
                removeEventListener: jest.fn()
            }));

            const {result} = renderHook(() => useTheme());

            act(() => {
                result.current.setTheme('auto');
            });

            expect(result.current.resolvedTheme).toBe('dark');
        });
    });
});

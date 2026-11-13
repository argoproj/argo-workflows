import type {ReactNode} from 'react';

type Modules = {
    React: typeof import('react');
    rtl: typeof import('@testing-library/react');
    argo: typeof import('argo-ui/v2/shared/context/theme');
    sync: typeof import('./theme-sync');
};

function load({stored, systemDark = false, noMatchMedia = false}: {stored?: string; systemDark?: boolean; noMatchMedia?: boolean}) {
    localStorage.clear();
    if (stored !== undefined) {
        localStorage.setItem('theme', stored);
    }
    const listeners = new Set<() => void>();
    const media = {
        matches: systemDark,
        addEventListener: (_: string, listener: () => void) => listeners.add(listener),
        removeEventListener: (_: string, listener: () => void) => listeners.delete(listener)
    };
    window.matchMedia = noMatchMedia ? undefined : jest.fn().mockReturnValue(media);
    document.body.className = '';
    let modules: Modules;
    jest.isolateModules(() => {
        modules = {React: require('react'), rtl: require('@testing-library/react'), argo: require('argo-ui/v2/shared/context/theme'), sync: require('./theme-sync')};
    });
    const {React, rtl, argo, sync} = modules!;
    const wrapper = ({children}: {children?: ReactNode}) => <argo.ThemeProvider>{children}</argo.ThemeProvider>;
    const render = () =>
        rtl.renderHook(
            () => {
                const theme = sync.useThemeSync();
                return {theme, ctx: React.useContext(argo.ThemeContext)};
            },
            {wrapper}
        );
    return {
        render,
        Theme: argo.Theme,
        rtl,
        setSystemDark: (value: boolean) =>
            rtl.act(() => {
                media.matches = value;
                listeners.forEach(listener => listener());
            })
    };
}

describe('useThemeSync', () => {
    afterEach(() => jest.restoreAllMocks());

    it('defaults to light when the OS is light and nothing is stored', () => {
        const {render, Theme} = load({systemDark: false});
        const {result} = render();
        expect(result.current.theme).toBe(Theme.Light);
        expect(document.body.classList.contains('theme-dark')).toBe(false);
    });

    it('defaults to dark when the OS is dark and nothing is stored', () => {
        const {render, Theme} = load({systemDark: true});
        const {result} = render();
        expect(result.current.theme).toBe(Theme.Dark);
        expect(document.body.classList.contains('theme-dark')).toBe(true);
        expect(JSON.parse(localStorage.getItem('theme')!)).toBe('dark');
    });

    it('follows the OS while the user has not chosen', () => {
        const {render, setSystemDark, Theme} = load({systemDark: false});
        const {result} = render();
        setSystemDark(true);
        expect(result.current.theme).toBe(Theme.Dark);
        expect(document.body.classList.contains('theme-dark')).toBe(true);
        setSystemDark(false);
        expect(result.current.theme).toBe(Theme.Light);
        expect(document.body.classList.contains('theme-dark')).toBe(false);
    });

    it('keeps a stored theme over the OS setting', () => {
        const {render, setSystemDark, Theme} = load({stored: '"light"', systemDark: true});
        const {result} = render();
        expect(result.current.theme).toBe(Theme.Light);
        setSystemDark(false);
        setSystemDark(true);
        expect(result.current.theme).toBe(Theme.Light);
    });

    it('stops following the OS once the user picks a theme', () => {
        const {render, setSystemDark, rtl, Theme} = load({systemDark: false});
        const {result} = render();
        rtl.act(() => result.current.ctx.set(Theme.Dark));
        expect(result.current.theme).toBe(Theme.Dark);
        setSystemDark(true);
        setSystemDark(false);
        expect(result.current.theme).toBe(Theme.Dark);
        expect(JSON.parse(localStorage.getItem('theme')!)).toBe('dark');
    });

    it('works when the browser has no matchMedia', () => {
        const {render, Theme} = load({noMatchMedia: true});
        const {result} = render();
        expect(result.current.theme).toBe(Theme.Light);
    });

    it('treats unreadable storage as nothing stored', () => {
        const {render, Theme} = load({systemDark: true});
        jest.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
            throw new Error('denied');
        });
        const {result} = render();
        expect(result.current.theme).toBe(Theme.Dark);
    });
});

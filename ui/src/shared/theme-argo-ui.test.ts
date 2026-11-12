import type * as ThemeModule from './theme';

type Theme = typeof ThemeModule;

describe('theme storage shared with argo-ui', () => {
    beforeEach(() => localStorage.clear());

    it.each(['system', 'light', 'dark'] as const)('argo-ui can load the stored %s preference', preference => {
        let theme: Theme;
        jest.isolateModules(() => {
            theme = require('./theme');
        });
        theme!.setThemePreference(preference);
        jest.isolateModules(() => {
            expect(() => require('argo-ui/v2/shared/context/theme')).not.toThrow();
        });
    });
});

import {render, screen} from '@testing-library/react';
import * as React from 'react';

jest.mock('react-monaco-editor', () => ({
    __esModule: true,
    default: ({theme}: {theme: string}) => <div data-testid='monaco' data-theme={theme} />
}));

function renderEditor(stored?: string) {
    localStorage.clear();
    if (stored !== undefined) {
        localStorage.setItem('theme', stored);
    }
    let argo: typeof import('argo-ui/v2/shared/context/theme');
    let editor: typeof import('./suspense-monaco-editor');
    jest.isolateModules(() => {
        argo = require('argo-ui/v2/shared/context/theme');
        editor = require('./suspense-monaco-editor');
    });
    const {ThemeProvider} = argo!;
    const {SuspenseMonacoEditor} = editor!;
    const ui = (
        <ThemeProvider>
            <SuspenseMonacoEditor />
        </ThemeProvider>
    );
    return render(ui);
}

describe('SuspenseMonacoEditor', () => {
    it('uses the light editor theme by default', async () => {
        renderEditor();
        expect((await screen.findByTestId('monaco')).getAttribute('data-theme')).toBe('vs');
    });

    it('uses the dark editor theme when the app theme is dark', async () => {
        renderEditor('"dark"');
        expect((await screen.findByTestId('monaco')).getAttribute('data-theme')).toBe('vs-dark');
    });
});

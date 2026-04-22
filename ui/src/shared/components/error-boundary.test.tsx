import {render, screen} from '@testing-library/react';
import React from 'react';

import ErrorBoundary from './error-boundary';

const CHUNK_RELOAD_KEY = 'argo-chunk-reload-attempted';

function Thrower({error}: {error: Error}): JSX.Element {
    throw error;
}

function makeChunkLoadError(): Error {
    const err = new Error('Loading chunk 775 failed.');
    err.name = 'ChunkLoadError';
    return err;
}

describe('ErrorBoundary', () => {
    let reloadMock: jest.Mock;
    let originalLocation: Location;
    let consoleErrorSpy: jest.SpyInstance;

    beforeEach(() => {
        sessionStorage.clear();
        reloadMock = jest.fn();
        originalLocation = window.location;
        Object.defineProperty(window, 'location', {
            configurable: true,
            value: {...originalLocation, reload: reloadMock}
        });
        // React logs caught errors via console.error; suppress so test output stays readable.
        consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation(() => undefined);
    });

    afterEach(() => {
        Object.defineProperty(window, 'location', {
            configurable: true,
            value: originalLocation
        });
        consoleErrorSpy.mockRestore();
    });

    it('reloads once when a lazy chunk fails to load', () => {
        render(
            <ErrorBoundary>
                <Thrower error={makeChunkLoadError()} />
            </ErrorBoundary>
        );

        expect(reloadMock).toHaveBeenCalledTimes(1);
        expect(sessionStorage.getItem(CHUNK_RELOAD_KEY)).toBe('1');
    });

    it('detects ChunkLoadError by message even when error.name is generic', () => {
        const err = new Error('Loading chunk 42 failed. (missing: https://x/42.abc.js)');

        render(
            <ErrorBoundary>
                <Thrower error={err} />
            </ErrorBoundary>
        );

        expect(reloadMock).toHaveBeenCalledTimes(1);
    });

    it('does not reload again if a chunk fails after a previous reload attempt', () => {
        sessionStorage.setItem(CHUNK_RELOAD_KEY, '1');

        render(
            <ErrorBoundary>
                <Thrower error={makeChunkLoadError()} />
            </ErrorBoundary>
        );

        expect(reloadMock).not.toHaveBeenCalled();
        expect(screen.getByRole('heading', {level: 3})).toHaveTextContent('Loading chunk 775 failed');
    });

    it('clears the reload flag and renders the error panel for non-chunk errors', () => {
        sessionStorage.setItem(CHUNK_RELOAD_KEY, '1');

        render(
            <ErrorBoundary>
                <Thrower error={new Error('Something else broke')} />
            </ErrorBoundary>
        );

        expect(reloadMock).not.toHaveBeenCalled();
        expect(sessionStorage.getItem(CHUNK_RELOAD_KEY)).toBeNull();
        expect(screen.getByRole('heading', {level: 3})).toHaveTextContent('Something else broke');
    });

    it('renders children unchanged when no error is thrown', () => {
        render(
            <ErrorBoundary>
                <div>healthy child</div>
            </ErrorBoundary>
        );

        expect(screen.getByText('healthy child')).toBeInTheDocument();
        expect(reloadMock).not.toHaveBeenCalled();
    });
});

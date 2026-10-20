import {render, screen} from '@testing-library/react';
import * as React from 'react';

import {ChunkLoadErrorBoundary} from './chunk-load-error-boundary';

beforeEach(() => jest.spyOn(console, 'error').mockImplementation(() => {}));
afterEach(() => jest.restoreAllMocks());

describe('ChunkLoadErrorBoundary', () => {
    it('renders children normally', () => {
        render(
            <ChunkLoadErrorBoundary>
                <div>Test</div>
            </ChunkLoadErrorBoundary>
        );
        expect(screen.getByText('Test')).toBeTruthy();
    });

    it('catches chunk errors', () => {
        const ThrowError = () => {
            throw new Error('Loading chunk 775 failed');
        };
        render(
            <ChunkLoadErrorBoundary>
                <ThrowError />
            </ChunkLoadErrorBoundary>
        );
        expect(screen.getByText(/Reloading/i)).toBeTruthy();
    });
});

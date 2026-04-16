import * as React from 'react';

import {lazyImport} from './lazy-import';

const MockComponent: React.FC = () => null;

describe('lazyImport', () => {
    it('imports successfully', async () => {
        const mockImport = jest.fn().mockResolvedValue({default: MockComponent});
        const result = await lazyImport(mockImport);
        expect(result.default).toBe(MockComponent);
    });
});

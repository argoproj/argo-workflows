import {render, screen, waitFor} from '@testing-library/react';
import React from 'react';

import requests from '../shared/services/requests';
import {UserInfo} from './user-info';

jest.mock('../shared/services/requests');
jest.mock('argo-ui/src/components/page/page', () => ({
    Page: ({children}: any) => <div>{children}</div>
}));

describe('UserInfo', () => {
    beforeEach(() => {
        jest.clearAllMocks();
        const base = document.createElement('base');
        base.setAttribute('href', '/');
        document.head.appendChild(base);
    });

    afterEach(() => {
        document.querySelector('base')?.remove();
    });

    // Verify that groups are displayed in alphabetical order regardless of the API response order
    test('displays groups sorted alphabetically', async () => {
        // API returns groups in unsorted order
        jest.spyOn(requests, 'get').mockResolvedValue({body: {groups: ['zeta', 'alpha', 'beta']}} as any);

        render(<UserInfo />);

        // Groups should be rendered as sorted: "alpha, beta, zeta"
        await waitFor(() => {
            expect(screen.getByText('alpha, beta, zeta')).toBeInTheDocument();
        });
    });
});

import {render, waitFor} from '@testing-library/react';
import {createMemoryHistory} from 'history';
import React from 'react';
import {NEVER} from 'rxjs';

import {App} from '../app';
import requests from '../shared/services/requests';

jest.mock('../shared/services/requests');

describe('EventFlowPage', () => {
    beforeEach(() => {
        jest.clearAllMocks();
        jest.spyOn(requests, 'get').mockResolvedValue({body: {metadata: {}, items: []}} as any);
        jest.spyOn(requests, 'post').mockReturnValue({send: jest.fn().mockResolvedValue('')} as any);
        jest.spyOn(requests, 'loadEventSource').mockReturnValue(NEVER);
    });

    it('limits the workflow list, which includes archived workflows', async () => {
        const history = createMemoryHistory();
        history.push('/event-flow/argo');

        render(<App history={history} />);

        await waitFor(() => {
            expect(requests.get).toHaveBeenCalledWith(expect.stringMatching(/^api\/v1\/workflows\/argo\?.*listOptions\.limit=500/));
        });
    });
});

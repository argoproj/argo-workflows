import {fireEvent, render, screen, waitFor} from '@testing-library/react';
import {createMemoryHistory} from 'history';
import React from 'react';
import {NEVER} from 'rxjs';

import {App} from '../app';
import requests from '../shared/services/requests';

jest.mock('../shared/services/requests');

// Stand-in namespace filter: one button per namespace, plus one for all.
jest.mock('../shared/components/namespace-filter', () => ({
    NamespaceFilter: ({onChange}: {onChange: (namespace: string) => void}) => (
        <>
            {['', 'ns-a', 'ns-b', 'ns-c'].map(ns => (
                <button key={ns} onClick={() => onChange(ns)}>
                    {`namespace:${ns || 'all'}`}
                </button>
            ))}
        </>
    )
}));

// Rows render their title as plain text; the lazy-loaded markdown renderer is not needed here.
jest.mock('../shared/components/suspense-react-markdown-gfm', () => ({
    SuspenseReactMarkdownGfm: ({markdown}: {markdown: string}) => <span>{markdown}</span>
}));

// The create panel's code editor is lazy-loaded and not under test.
jest.mock('./cron-workflow-creator', () => ({CronWorkflowCreator: (): null => null}));

const namespaces = ['ns-a', 'ns-b', 'ns-c'];
const names = ['build', 'release', 'sync', 'test', 'update'];

// The same five CronWorkflow names in every namespace.
function cronWorkflows(namespace: string) {
    return namespaces
        .filter(ns => !namespace || ns === namespace)
        .flatMap(ns => names.map(name => ({metadata: {name, namespace: ns, creationTimestamp: '2026-01-01T00:00:00Z'}, spec: {schedules: ['0 * * * *'], workflowSpec: {}}})));
}

// Each rendered row as "<namespace>/<name>", read from its link.
function rows(container: HTMLElement): string[] {
    return Array.from(container.querySelectorAll('.cron-workflows-list__row-container a')).map(a => a.getAttribute('href').split('/').slice(-2).join('/'));
}

function expected(namespace: string): string[] {
    return cronWorkflows(namespace).map(t => `${t.metadata.namespace}/${t.metadata.name}`);
}

describe('CronWorkflowList', () => {
    beforeEach(() => {
        jest.clearAllMocks();
        localStorage.clear();
        jest.spyOn(requests, 'get').mockImplementation(((url: string) => {
            const match = url.match(/^api\/v1\/cron-workflows\/([^?/]*)\?/);
            if (match) {
                return Promise.resolve({body: {metadata: {}, items: cronWorkflows(match[1])}});
            }
            return Promise.resolve({body: {metadata: {}, items: []}});
        }) as any);
        jest.spyOn(requests, 'post').mockReturnValue({send: jest.fn().mockResolvedValue('')} as any);
        jest.spyOn(requests, 'loadEventSource').mockReturnValue(NEVER);
    });

    it('shows only the selected namespace when same-named CronWorkflows exist in several namespaces', async () => {
        const consoleError = jest.spyOn(console, 'error');
        const history = createMemoryHistory();
        history.push('/cron-workflows');
        const {container} = render(<App history={history} />);
        await waitFor(() => expect(rows(container)).toEqual(expected('')));

        for (const namespace of ['ns-a', 'ns-b', '', 'ns-c', 'ns-a']) {
            fireEvent.click(screen.getByText(`namespace:${namespace || 'all'}`));
            await waitFor(() => expect(rows(container)).toEqual(expected(namespace)));
        }

        expect(consoleError.mock.calls.map(([msg]) => String(msg)).filter(m => m.includes('Encountered two children with the same key'))).toEqual([]);
    });
});

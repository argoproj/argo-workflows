import * as React from 'react';

import {request} from './api';
import {definitionKinds} from './definition-editor';

const prefix = 'parameters.workflows.example.io/definition-';

// Publications are ordinary definitions with a new immutable name. Opening,
// comparing or cloning one never submits or retries a Workflow.
export function DefinitionPublications({namespace, resource, document, onSelected}: {namespace: string; resource: string; document: any; onSelected: (document: any) => void}) {
    const [publishing, setPublishing] = React.useState(false);
    const [name, setName] = React.useState(document.metadata.name + '-v1');
    const [message, setMessage] = React.useState('');
    const [items, setItems] = React.useState<any[]>([]);
    const [cursor, setCursor] = React.useState('');
    const [loaded, setLoaded] = React.useState(false);
    const [comparison, setComparison] = React.useState<any>();
    const [busy, setBusy] = React.useState(false);
    const [error, setError] = React.useState('');
    const active = React.useRef(true);
    React.useEffect(() => {
        active.current = true;
        return () => {
            active.current = false;
        };
    }, []);
    if (!definitionKinds[resource] || !document.metadata.uid) return null;
    const family = document.metadata.labels?.[prefix + 'family'] || document.metadata.uid;
    const action = async (fn: () => Promise<void>) => {
        setBusy(true);
        setError('');
        try {
            await fn();
        } catch (e) {
            if (active.current) setError(e.response?.body?.message || e.message);
        } finally {
            if (active.current) setBusy(false);
        }
    };
    const load = (more = false) =>
        action(async () => {
            const page = await request(
                namespace,
                'definition-publications/' + resource + '?family=' + encodeURIComponent(family) + (more ? '&continue=' + encodeURIComponent(cursor) : '')
            );
            if (!active.current) return;
            setItems(old => (more ? [...old, ...(page.items || [])] : page.items || []));
            setCursor(page.metadata?.continue || '');
            setLoaded(true);
        });
    const compare = (left: string, more = false) =>
        action(async () => {
            const page = await request(namespace, 'definition-diff/' + resource, 'post', {
                left,
                right: document.metadata.name,
                limit: 100,
                ...(more ? {continue: comparison.pagination.continue} : {})
            });
            if (active.current) setComparison((old: any) => (more ? {...page, changes: [...old.changes, ...page.changes]} : page));
        });
    return (
        <section aria-label='Definition publications'>
            <h4>Published definition versions</h4>
            <p>A publication preserves this definition under a new immutable name. References are resolved and frozen separately when a Workflow is submitted.</p>
            <button
                disabled={busy}
                onClick={() => {
                    setPublishing(true);
                    setError('');
                }}>
                Publish named version
            </button>
            <button disabled={busy} onClick={() => load()}>
                View published versions
            </button>
            {publishing && (
                <div>
                    <label>
                        Version name <input aria-label='Published definition name' value={name} disabled={busy} onChange={e => setName(e.target.value)} />
                    </label>
                    <label>
                        Publication note <input aria-label='Publication note' value={message} maxLength={2048} disabled={busy} onChange={e => setMessage(e.target.value)} />
                    </label>
                    <button
                        disabled={busy || !name || name === document.metadata.name}
                        onClick={() =>
                            action(async () => {
                                const result = await request(namespace, 'definition-publications/' + resource, 'post', {
                                    source: document.metadata.name,
                                    uid: document.metadata.uid,
                                    sourceSpec: document.spec,
                                    name,
                                    message
                                });
                                if (active.current) onSelected(result);
                            })
                        }>
                        Confirm publish definition
                    </button>
                    <button disabled={busy} onClick={() => setPublishing(false)}>
                        Cancel publication
                    </button>
                </div>
            )}
            {busy && <p role='status'>Loading definition versions…</p>}
            {error && <p role='alert'>{error}</p>}
            {loaded && !items.length && <p>No published versions in this definition family.</p>}
            {items.map(item => (
                <div key={item.metadata.uid}>
                    <span>
                        {item.metadata.name} · {item.metadata.annotations?.[prefix + 'published-at']} · {item.metadata.annotations?.[prefix + 'message']}
                    </span>
                    <button
                        disabled={busy}
                        onClick={() =>
                            action(async () => {
                                const result = await request(namespace, 'catalog/' + resource + '?name=' + encodeURIComponent(item.metadata.name));
                                if (result.metadata.uid !== item.metadata.uid) throw new Error('Published version was replaced; refresh history.');
                                if (active.current) onSelected(result);
                            })
                        }>
                        Open version {item.metadata.name}
                    </button>
                    <button disabled={busy} onClick={() => compare(item.metadata.name)}>
                        Compare {item.metadata.name} with selected definition
                    </button>
                </div>
            ))}
            {cursor && (
                <button disabled={busy} onClick={() => load(true)}>
                    Load more definition versions
                </button>
            )}
            {comparison && (
                <div aria-label='Definition version comparison'>
                    <p>
                        {comparison.left} → {comparison.right} · {comparison.pagination.total} changes
                    </p>
                    <pre>{JSON.stringify(comparison.changes, null, 2)}</pre>
                    {comparison.pagination.continue && (
                        <button disabled={busy} onClick={() => compare(comparison.left, true)}>
                            Load more definition changes
                        </button>
                    )}
                </div>
            )}
            <p>
                To restore a published version, open it and choose “Clone as new definition”. Select that new name explicitly in a binding; existing definitions and executions stay
                unchanged.
            </p>
        </section>
    );
}

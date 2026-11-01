import * as React from 'react';
import {useEffect, useRef, useState} from 'react';

import {request} from './api';
import {RevisionDiff} from './revision-diff';

// Keep rendering bounded too. Page navigation reads the same immutable pair
// again under current permissions; it never saves or applies parameter values.
export function PagedRevisionDiff({namespace, comparison, execution = false}: {namespace: string; comparison: any; execution?: boolean}) {
    const [page, setPage] = useState(comparison);
    const [index, setIndex] = useState(0);
    const [cursors, setCursors] = useState(['']);
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState('');
    const generation = useRef(0);
    const pending = useRef(false);
    useEffect(() => {
        generation.current++;
        pending.current = false;
        setPage(comparison);
        setIndex(0);
        setCursors(['']);
        setBusy(false);
        setError('');
        return () => {
            generation.current++;
        };
    }, [namespace, comparison, execution]);
    const move = async (next: number) => {
        if (pending.current) return;
        const cursor = next > index ? page.pagination.continue : cursors[next];
        const current = generation.current;
        pending.current = true;
        setBusy(true);
        setError('');
        try {
            const body = execution
                ? {source: comparison.source, revision: comparison.parameters.right, snapshot: comparison.pagination.snapshot, limit: page.pagination.limit, continue: cursor}
                : {left: comparison.left, right: comparison.right, limit: page.pagination.limit, continue: cursor};
            const value = await request(namespace, execution ? 'execution-diff' : 'diff', 'post', body);
            if (current !== generation.current) return;
            setPage(value);
            setIndex(next);
            setCursors(old => {
                const updated = [...old];
                updated[next] = cursor;
                return updated;
            });
        } catch (e) {
            if (current === generation.current) setError(e.response?.body?.message || e.message);
        } finally {
            if (current === generation.current) {
                pending.current = false;
                setBusy(false);
            }
        }
    };
    const visible = execution ? {...page.parameters, layers: page.layers} : page;
    const count =
        (visible.changes?.length || 0) +
        (visible.modelChanges?.length || 0) +
        (visible.sourceChanges?.length || 0) +
        (visible.layers || []).reduce((total: number, layer: any) => total + layer.changes.length, 0);
    return (
        <>
            {page.pagination && (
                <nav aria-label='Diff pages'>
                    <p>
                        Changes {count ? page.pagination.offset + 1 : 0}–{page.pagination.offset + count} of {page.pagination.total}. Filters apply to this page.
                    </p>
                    <button disabled={busy || index === 0} onClick={() => move(index - 1)}>
                        Previous diff page
                    </button>
                    <button disabled={busy || !page.pagination.continue} onClick={() => move(index + 1)}>
                        Next diff page
                    </button>
                    {busy && <span role='status'>Loading diff page…</span>}
                    {error && <p role='alert'>{error}</p>}
                </nav>
            )}
            <RevisionDiff comparison={visible} />
        </>
    );
}

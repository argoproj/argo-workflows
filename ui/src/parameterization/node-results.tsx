import * as React from 'react';
import {useEffect, useRef, useState} from 'react';

import {NodeStatus, Workflow} from '../shared/models';
import {request} from './api';

interface Column {
    title: string;
    path: string;
    format?: 'text' | 'link' | 'status';
}
interface View {
    title: string;
    type: 'summary' | 'table' | 'tree' | 'text';
    path?: string;
    columns?: Column[];
    childrenPath?: string;
}
interface Result {
    name: string;
    title: string;
    scope: string;
    available: boolean;
    error?: string;
    observedAt?: string;
    stale?: boolean;
    value?: any;
    views: View[];
}
interface Results {
    uid: string;
    nodeId: string;
    invocation: string;
    items: Result[];
}

function pointer(value: any, path = ''): any {
    if (!path) return value;
    if (!path.startsWith('/')) return undefined;
    return path
        .slice(1)
        .split('/')
        .reduce(
            (v, k) => (v != null && Object.prototype.hasOwnProperty.call(v, k.replace(/~1/g, '/').replace(/~0/g, '~')) ? v[k.replace(/~1/g, '/').replace(/~0/g, '~')] : undefined),
            value
        );
}
const text = (value: any) => (value === undefined ? '—' : typeof value === 'string' ? value : JSON.stringify(value));

function Cell({value, column}: {value: any; column: Column}) {
    const field = pointer(value, column.path);
    if (column.format === 'link' && typeof field === 'string') {
        try {
            const url = new URL(field);
            if (['https:', 'http:'].includes(url.protocol) && !url.username && !url.password)
                return (
                    <a href={url.href} target='_blank' rel='noopener noreferrer'>
                        {field}
                    </a>
                );
        } catch (_) {
            /* Untrusted report strings are displayed as text. */
        }
    }
    return <span>{text(field)}</span>;
}

export function ResultView({view, value}: {view: View; value: any}) {
    const [page, setPage] = useState(0);
    const source = pointer(value, view.path);
    const columns = view.columns || [];
    useEffect(() => setPage(0), [value, view.path]);
    if (source === undefined) return <p role='alert'>View data is unavailable.</p>;
    if (view.type === 'text') return <pre style={{whiteSpace: 'pre-wrap'}}>{text(source)}</pre>;
    if (view.type === 'summary')
        return (
            <dl>
                {columns.map((column, i) => (
                    <React.Fragment key={i}>
                        <dt>{column.title}</dt>
                        <dd>
                            <Cell value={source} column={column} />
                        </dd>
                    </React.Fragment>
                ))}
            </dl>
        );
    if (!Array.isArray(source)) return <p role='alert'>This view requires an array.</p>;
    const rows: Array<{value: any; depth: number}> = [];
    let truncated = false;
    const collect = (items: any[], depth: number) => {
        for (const item of items) {
            if (rows.length >= 1000 || depth > 8) {
                truncated = true;
                break;
            }
            rows.push({value: item, depth});
            if (view.type === 'tree') {
                const children = pointer(item, view.childrenPath);
                if (Array.isArray(children)) collect(children, depth + 1);
            }
        }
    };
    collect(source, 0);
    return (
        <>
            <table className='argo-table-list'>
                <thead>
                    <tr>
                        {columns.map((c, i) => (
                            <th key={i}>{c.title}</th>
                        ))}
                    </tr>
                </thead>
                <tbody>
                    {rows.slice(page * 50, (page + 1) * 50).map((row, i) => (
                        <tr key={i}>
                            {columns.map((c, j) => (
                                <td key={j} style={j === 0 ? {paddingLeft: row.depth * 16} : undefined}>
                                    <Cell value={row.value} column={c} />
                                </td>
                            ))}
                        </tr>
                    ))}
                </tbody>
            </table>
            {!rows.length && <p>No result rows.</p>}
            {rows.length > 50 && (
                <div>
                    <button disabled={page === 0} onClick={() => setPage(p => p - 1)}>
                        Previous rows
                    </button>
                    <span>
                        {' '}
                        Page {page + 1} / {Math.ceil(rows.length / 50)}{' '}
                    </span>
                    <button disabled={(page + 1) * 50 >= rows.length} onClick={() => setPage(p => p + 1)}>
                        Next rows
                    </button>
                </div>
            )}
            {truncated && <p role='status'>Display limited to 1,000 rows and 8 nested levels. Use the action artifact for the complete report.</p>}
        </>
    );
}

export function NodeResults({workflow, node}: {workflow: Workflow; node: NodeStatus}) {
    const [data, setData] = useState<Results>();
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const [revision, setRevision] = useState(0);
    const scope = `${workflow.metadata.uid}/${node.id}/${node.startedAt || ''}`;
    const active = useRef(scope);
    active.current = scope;
    const enabled = !!workflow.metadata.annotations?.['parameters.workflows.example.io/submission'];
    useEffect(() => {
        setData(undefined);
        setError('');
    }, [scope]);
    useEffect(() => {
        if (!enabled) return;
        let current = true;
        setLoading(true);
        setError('');
        const query = new URLSearchParams({uid: workflow.metadata.uid, nodeId: node.id});
        request(workflow.metadata.namespace, `node-results/${encodeURIComponent(workflow.metadata.name)}?${query.toString()}`)
            .then((value: Results) => {
                if (!current || active.current !== scope) return;
                if (value.uid !== workflow.metadata.uid || value.nodeId !== node.id) throw new Error('Result identity changed; reopen the node.');
                setData(value);
            })
            .catch(e => {
                if (!current || active.current !== scope) return;
                if (e.status === 403 || e.status === 401) setData(undefined);
                setError(e.response?.body?.message || e.message || 'Results are temporarily unavailable.');
            })
            .finally(() => {
                if (current && active.current === scope) setLoading(false);
            });
        return () => {
            current = false;
        };
    }, [enabled, scope, revision, node.phase]);
    if (!enabled || (!loading && !error && !data?.items.length)) return null;
    return (
        <section aria-label='Action results'>
            <h4>Action results</h4>
            <button disabled={loading} onClick={() => setRevision(n => n + 1)}>
                Refresh results
            </button>
            <p>Reads the saved output or the execution adapter’s observation. Does not rerun this action.</p>
            {loading && <p role='status'>Reading results…</p>}
            {error && (
                <p role='alert'>
                    {error}
                    {data && ' Previously loaded results are retained; they are not a fresh observation.'}
                </p>
            )}
            {data?.items.map(item => (
                <article key={item.name} aria-label={item.title || item.name}>
                    <h5>{item.title || item.name}</h5>
                    <p>
                        {item.scope === 'LiveObservation' ? 'Current observation (separate from the execution outcome)' : 'Saved execution output'}
                        {item.observedAt && ` · ${item.observedAt}`}
                    </p>
                    {item.stale && <p role='status'>Observation is stale; its controller has not provided a recent update.</p>}
                    {!item.available ? (
                        <p role='alert'>{item.error || 'Result unavailable'}</p>
                    ) : (
                        item.views.map((view, i) => (
                            <section key={i}>
                                {view.title && <h6>{view.title}</h6>}
                                <ResultView view={view} value={item.value} />
                            </section>
                        ))
                    )}
                </article>
            ))}
        </section>
    );
}

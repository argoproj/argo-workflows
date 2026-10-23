import * as React from 'react';
import {useEffect, useState} from 'react';

import {uiUrl} from '../shared/base';
import {request} from './api';

// Discovery is optional: an unextended Argo server and unbound templates retain
// their native submit entry. Kubernetes namespace/RBAC still governs the list.
export function BindingLinks({namespace, workflowTemplate, clusterScope = false}: {namespace: string; workflowTemplate: string; clusterScope?: boolean}) {
    const [bindings, setBindings] = useState<any[]>([]);
    const [error, setError] = useState('');
    useEffect(() => {
        let active = true;
        setBindings([]);
        setError('');
        (async () => {
            try {
                let cursor = '';
                const seen = new Set<string>();
                const matched: any[] = [];
                do {
                    const page = await request(namespace, 'bindings' + (cursor ? '?continue=' + encodeURIComponent(cursor) : ''));
                    if (!active) return;
                    matched.push(...(page.items || []).filter((item: any) => item.spec?.workflowTemplate === workflowTemplate && !!item.spec?.clusterScope === clusterScope));
                    cursor = page.metadata?.continue || '';
                    if (cursor && seen.has(cursor)) throw new Error('Binding pagination did not advance.');
                    seen.add(cursor);
                } while (cursor);
                setBindings(matched);
            } catch (e) {
                if (active && (e.status || e.response?.status) !== 404) setError('Structured parameter bindings could not be loaded.');
            }
        })();
        return () => {
            active = false;
        };
    }, [namespace, workflowTemplate, clusterScope]);
    if (!bindings.length && !error) return null;
    return (
        <section className='white-box' aria-label='Structured parameter bindings'>
            <h4>Structured parameter bindings</h4>
            {error && <p role='status'>{error}</p>}
            {bindings.length > 0 && <p>Choose the input model and form for this template. Opening a binding does not submit a Workflow.</p>}
            <ul>
                {bindings.map(binding => (
                    <li key={binding.metadata.name}>
                        <a href={uiUrl('parameters/' + encodeURIComponent(namespace) + '?binding=' + encodeURIComponent(binding.metadata.name))}>{binding.metadata.name}</a>
                        {' · '}
                        {binding.spec.parameterTemplate}
                        {binding.spec.form ? ' · ' + binding.spec.form : ''}
                    </li>
                ))}
            </ul>
        </section>
    );
}

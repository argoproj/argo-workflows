import * as React from 'react';
import {useEffect, useRef, useState} from 'react';
import * as YAML from 'yaml';

import {SuspenseMonacoEditor} from '../shared/components/suspense-monaco-editor';
import {request} from './api';
import {DefinitionEditor, definitionKinds} from './definition-editor';
import {DefinitionPublications} from './definition-publications';
import {DefinitionValidation} from './definition-validation';

export function ResourceCatalog({namespace}: {namespace: string}) {
    const [types, setTypes] = useState<any[]>([]);
    const [type, setType] = useState('parametertemplates');
    const [items, setItems] = useState<any[]>([]);
    const [cursor, setCursor] = useState('');
    const [selected, setSelected] = useState<any>();
    const [error, setError] = useState('');
    const [busy, setBusy] = useState(false);
    const generation = useRef(0);
    const detailGeneration = useRef(0);
    const [detailBusy, setDetailBusy] = useState(false);
    const load = async (more = false) => {
        const current = ++generation.current;
        setBusy(true);
        setError('');
        if (!more) {
            detailGeneration.current++;
            setDetailBusy(false);
            setItems([]);
            setSelected(undefined);
        }
        try {
            const page = await request(namespace, 'catalog/' + type + '?view=metadata' + (more ? '&continue=' + encodeURIComponent(cursor) : ''));
            if (current !== generation.current) return;
            setItems(old => (more ? [...old, ...(page.items || [])] : page.items || []));
            setCursor(page.metadata?.continue || '');
        } catch (e) {
            if (current === generation.current) setError(e.response?.body?.message || e.message);
        } finally {
            if (current === generation.current) setBusy(false);
        }
    };
    const select = async (item: any) => {
        const current = ++detailGeneration.current;
        setSelected(undefined);
        setDetailBusy(true);
        setError('');
        try {
            const document = await request(namespace, 'catalog/' + type + '?name=' + encodeURIComponent(item.metadata.name));
            if (current !== detailGeneration.current) return;
            if (item.metadata.uid && document.metadata.uid !== item.metadata.uid) throw new Error('Resource was replaced; refresh the list.');
            setSelected(document);
        } catch (e) {
            if (current === detailGeneration.current) setError(e.response?.body?.message || e.message);
        } finally {
            if (current === detailGeneration.current) setDetailBusy(false);
        }
    };
    useEffect(() => {
        let active = true;
        request(namespace, 'catalog')
            .then(data => {
                if (active) setTypes(data);
            })
            .catch(e => {
                if (active) setError(e.message);
            });
        return () => {
            active = false;
        };
    }, [namespace]);
    useEffect(() => {
        load();
        return () => {
            generation.current++;
            detailGeneration.current++;
        };
    }, [namespace, type]);
    return (
        <section aria-label='Parameter resource catalog'>
            <h3>Parameter resources</h3>
            <p>Browse definitions and execution receipts in this namespace. Definition edits use your Kubernetes permissions; execution receipts remain read-only.</p>
            <div className='parameter-toolbar'>
                <label>
                    Resource type{' '}
                    <select aria-label='Parameter resource type' value={type} onChange={e => setType(e.target.value)}>
                        {types.map(t => (
                            <option key={t.resource} value={t.resource}>
                                {t.kind}
                            </option>
                        ))}
                    </select>
                </label>
                <button disabled={busy} onClick={() => load()}>
                    Refresh resources
                </button>
                {definitionKinds[type] && (
                    <button
                        onClick={() => {
                            detailGeneration.current++;
                            setDetailBusy(false);
                            setError('');
                            setSelected({kind: definitionKinds[type], metadata: {namespace}, spec: {}});
                        }}>
                        Create definition
                    </button>
                )}
            </div>
            <p>{types.find(t => t.resource === type)?.description}</p>
            {error && (
                <div role='alert' className='error'>
                    {error}
                </div>
            )}
            {busy && <p role='status'>Loading resources…</p>}
            {!busy && !error && !items.length && <p>No resources of this type in {namespace}.</p>}
            <div className='parameter-resource-grid'>
                <div>
                    <table>
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Resource version</th>
                            </tr>
                        </thead>
                        <tbody>
                            {items.map(item => (
                                <tr key={item.metadata.uid || item.metadata.name}>
                                    <td>
                                        <button aria-pressed={selected?.metadata.name === item.metadata.name} onClick={() => select(item)}>
                                            {item.metadata.name}
                                        </button>
                                    </td>
                                    <td>{item.metadata.resourceVersion}</td>
                                </tr>
                            ))}
                        </tbody>
                    </table>
                    {cursor && (
                        <button disabled={busy} onClick={() => load(true)}>
                            Load more resources
                        </button>
                    )}
                </div>
                {detailBusy && <p role='status'>Loading selected document…</p>}
                {selected && (
                    <div>
                        <h4>
                            {selected.kind} · {selected.metadata.name}
                        </h4>
                        <p>Namespace: {selected.metadata.namespace}</p>
                        {definitionKinds[type] && <DefinitionValidation document={selected} />}
                        <DefinitionPublications
                            key={'versions/' + namespace + '/' + type + '/' + selected.metadata.uid + '/' + selected.metadata.resourceVersion}
                            namespace={namespace}
                            resource={type}
                            document={selected}
                            onSelected={document => {
                                setSelected(document);
                                setItems(old => (old.some(item => item.metadata.name === document.metadata.name) ? old : [{metadata: document.metadata}, ...old]));
                            }}
                        />
                        <DefinitionEditor
                            key={namespace + '/' + type + '/' + (selected.metadata.uid || selected.metadata.name || 'new') + '/' + selected.metadata.resourceVersion}
                            namespace={namespace}
                            resource={type}
                            document={selected}
                            onSaved={document => {
                                setSelected(document);
                                setItems(old =>
                                    old.some(item => item.metadata.name === document.metadata.name)
                                        ? old.map(item => (item.metadata.name === document.metadata.name ? {metadata: document.metadata} : item))
                                        : [{metadata: document.metadata}, ...old]
                                );
                            }}
                        />
                        <SuspenseMonacoEditor
                            height={520}
                            language='yaml'
                            value={YAML.stringify(selected)}
                            options={{readOnly: true, folding: true, automaticLayout: true, minimap: {enabled: false}}}
                        />
                    </div>
                )}
            </div>
        </section>
    );
}

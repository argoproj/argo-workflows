import * as React from 'react';
import * as YAML from 'yaml';

import {request} from './api';

export const definitionKinds: {[resource: string]: string} = {
    parametertemplates: 'ParameterTemplate',
    workflowforms: 'WorkflowForm',
    workflowparameterbindings: 'WorkflowParameterBinding',
    actiondefinitions: 'ActionDefinition',
    actionresulttemplates: 'ActionResultTemplate',
    workflowactionbindings: 'WorkflowActionBinding'
};

function canonical(value: any): any {
    if (Array.isArray(value)) return value.map(canonical);
    if (value && typeof value === 'object')
        return Object.fromEntries(
            Object.keys(value)
                .sort()
                .map(key => [key, canonical(value[key])])
        );
    return value;
}

// Edit spec only; preserve Kubernetes metadata, status and immutable receipts.
export function DefinitionEditor({namespace, resource, document, onSaved}: {namespace: string; resource: string; document: any; onSaved: (document: any) => void}) {
    const [editing, setEditing] = React.useState(!document.metadata.uid);
    const [cloning, setCloning] = React.useState(false);
    const [name, setName] = React.useState(document.metadata.name || '');
    const [text, setText] = React.useState(YAML.stringify(document.spec || {}));
    const [message, setMessage] = React.useState('');
    const [error, setError] = React.useState('');
    const [busy, setBusy] = React.useState(false);
    const active = React.useRef(true);
    React.useEffect(() => {
        active.current = true;
        return () => {
            active.current = false;
        };
    }, []);
    if (!definitionKinds[resource]) return null;
    const immutable =
        !!document.metadata.annotations?.['parameters.workflows.example.io/definition-digest'] || resource === 'actiondefinitions' || resource === 'workflowactionbindings';
    const updating = !!document.metadata.uid && !cloning;
    const write = async (validateOnly: boolean) => {
        setMessage('');
        setError('');
        let spec: any;
        try {
            spec = YAML.parse(text, {uniqueKeys: true, maxAliasCount: 30});
            if (!spec || typeof spec !== 'object' || Array.isArray(spec)) throw new Error('Definition spec must be an object.');
        } catch (e) {
            setError(e.message);
            return;
        }
        setBusy(true);
        try {
            const put = (resourceVersion: string) =>
                request(namespace, 'definitions/' + resource + '?name=' + encodeURIComponent(name), 'put', {
                    spec,
                    uid: updating ? document.metadata.uid : '',
                    resourceVersion,
                    validateOnly
                });
            let result: any;
            try {
                result = await put(updating ? document.metadata.resourceVersion : '');
            } catch (conflict) {
                if (!updating || (conflict.status || conflict.response?.status) !== 409 || !active.current) throw conflict;
                const current = await request(namespace, 'catalog/' + resource + '?name=' + encodeURIComponent(name));
                if (!active.current) return;
                // One bounded retry is safe only for status/metadata-only changes.
                // A replacement or intervening spec generation retains the conflict.
                if (
                    current.metadata.uid !== document.metadata.uid ||
                    current.metadata.generation !== document.metadata.generation ||
                    current.metadata.resourceVersion === document.metadata.resourceVersion ||
                    JSON.stringify(canonical(current.spec)) !== JSON.stringify(canonical(document.spec))
                )
                    throw conflict;
                result = await put(current.metadata.resourceVersion);
            }
            if (!active.current) return;
            if (validateOnly) setMessage('Installed CRD schema validation passed. Execution references and values are checked during preview.');
            else {
                onSaved(result);
                setEditing(false);
                setMessage('Definition saved. Existing execution snapshots are unchanged.');
            }
        } catch (e) {
            if (active.current) setError(e.response?.body?.message || e.message);
        } finally {
            if (active.current) setBusy(false);
        }
    };
    return (
        <section aria-label='Definition editor'>
            {!editing && (
                <>
                    {immutable ? (
                        <p>This definition is immutable. Clone it into a new name and explicitly update its binding.</p>
                    ) : (
                        <button
                            onClick={() => {
                                setCloning(false);
                                setName(document.metadata.name);
                                setEditing(true);
                                setText(YAML.stringify(document.spec || {}));
                                setMessage('');
                                setError('');
                            }}>
                            Edit definition spec
                        </button>
                    )}
                    <button
                        onClick={() => {
                            setCloning(true);
                            setName(document.metadata.name + '-next');
                            setEditing(true);
                            setText(YAML.stringify(document.spec || {}));
                            setMessage('');
                            setError('');
                        }}>
                        Clone as new definition
                    </button>
                </>
            )}
            {editing && (
                <>
                    <p>Edits affect future previews and submissions. Saved parameter revisions and execution receipts retain their frozen definitions.</p>
                    <label>
                        Definition name <input aria-label='Definition name' value={name} disabled={busy || updating} onChange={e => setName(e.target.value)} />
                    </label>
                    <label>
                        Spec YAML
                        <textarea
                            aria-label='Definition spec YAML'
                            value={text}
                            disabled={busy}
                            rows={20}
                            onChange={e => {
                                setText(e.target.value);
                                setMessage('');
                                setError('');
                            }}
                        />
                    </label>
                    <button disabled={busy || !name} onClick={() => write(true)}>
                        Validate definition
                    </button>
                    <button disabled={busy || !name} onClick={() => write(false)}>
                        Save definition
                    </button>
                    {document.metadata.uid && (
                        <button
                            disabled={busy}
                            onClick={() => {
                                setEditing(false);
                                setMessage('');
                                setError('');
                            }}>
                            Cancel definition edit
                        </button>
                    )}
                </>
            )}
            {message && <p role='status'>{message}</p>}
            {error && <p role='alert'>{error}</p>}
        </section>
    );
}

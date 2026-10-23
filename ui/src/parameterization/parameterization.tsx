import {Page} from 'argo-ui/src/components/page/page';
import * as React from 'react';
import {useEffect, useRef, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import * as YAML from 'yaml';

import {uiUrl} from '../shared/base';
import {SuspenseMonacoEditor} from '../shared/components/suspense-monaco-editor';
import {request} from './api';
import {DeclaredForm} from './declared-form';
import {PagedRevisionDiff} from './paged-revision-diff';
import {ResourceCatalog} from './resource-catalog';

import './parameterization.scss';

export {request} from './api';

export function Parameterization({match, location}: RouteComponentProps<{namespace: string}>) {
    const ns = match.params.namespace;
    const requestedBinding = new URLSearchParams(location.search).get('binding');
    const sourceWorkflow = new URLSearchParams(location.search).get('sourceWorkflow');
    const sourceUID = new URLSearchParams(location.search).get('sourceUID');
    const [origin, setOrigin] = useState<any>();
    const [view, setView] = useState('inputs');
    const [bindings, setBindings] = useState<any[]>([]);
    const [secretInput, setSecretInput] = useState(false);
    const [binding, setBinding] = useState<any>();
    const [model, setModel] = useState<any>();
    const [form, setForm] = useState<any>({});
    const [formValid, setFormValid] = useState(true);
    const [setName, setSetName] = useState('my-parameters');
    const [draft, setDraft] = useState<any>();
    const [text, setText] = useState('{}\n');
    const [mode, setMode] = useState('form');
    const [history, setHistory] = useState<any[]>([]);
    const [cursor, setCursor] = useState('');
    const [left, setLeft] = useState('');
    const [right, setRight] = useState('');
    const [diff, setDiff] = useState<any>();
    const [preview, setPreview] = useState<any>();
    const [retryPlan, setRetryPlan] = useState<any>();
    const [message, setMessage] = useState('');
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState('');
    const [restoredFrom, setRestoredFrom] = useState('');
    const [workflow, setWorkflow] = useState('');
    const [submitKey, setSubmitKey] = useState('');
    const scope = useRef(0);
    const run = async (action: () => Promise<any>) => {
        setBusy(true);
        setError('');
        try {
            await action();
        } catch (e) {
            setError(e.response?.body?.message || e.message || String(e));
        } finally {
            setBusy(false);
        }
    };
    useEffect(() => {
        const current = ++scope.current;
        setBusy(true);
        setError('');
        setBinding(undefined);
        setOrigin(undefined);
        setSetName('my-parameters');
        setModel(undefined);
        setDraft(undefined);
        setHistory([]);
        setPreview(undefined);
        setRetryPlan(undefined);
        setDiff(undefined);
        setWorkflow('');
        setRestoredFrom('');
        setText('{}\n');
        setSubmitKey('');
        (async () => {
            try {
                const data = await request(ns, 'bindings');
                let available = data.items || [];
                let selected: any;
                let selectedModel: any;
                let selectedForm: any = {};
                let context: any;
                let comparison: any;
                let historyPage: any;
                if (sourceWorkflow || sourceUID) {
                    if (!sourceWorkflow || !sourceUID) throw new Error('The source execution link requires a Workflow name and UID.');
                    context = await request(ns, 'edit-context/' + encodeURIComponent(sourceWorkflow) + '?uid=' + encodeURIComponent(sourceUID));
                    if (requestedBinding && requestedBinding !== context.binding) throw new Error('The linked binding differs from the source execution.');
                    comparison = await request(ns, 'diff', 'post', {left: context.revision.metadata.name, right: context.set.spec.revision});
                    historyPage = await request(ns, 'history/' + encodeURIComponent(context.set.metadata.name) + '?view=summary');
                }
                const selectedBinding = context?.binding || requestedBinding;
                if (selectedBinding) {
                    selected = await request(ns, 'bindings/' + encodeURIComponent(selectedBinding));
                    if (!available.some((item: any) => item.metadata.name === selectedBinding)) available = [...available, selected];
                    if (context && (context.revision.spec.template !== selected.spec.parameterTemplate || context.set.spec.template !== selected.spec.parameterTemplate)) {
                        throw new Error('The binding parameter model has changed; migrate the parameter set before editing this execution.');
                    }
                    selectedModel = (await request(ns, 'models/' + selected.spec.parameterTemplate)).spec;
                    selectedForm = selected.spec.form ? (await request(ns, 'forms/' + selected.spec.form)).spec : {};
                }
                const capabilities = await request(ns, 'capabilities');
                if (current !== scope.current) return;
                setBindings(available);
                setBinding(selected);
                setModel(selectedModel);
                setForm(selectedForm);
                setSecretInput(!!capabilities.encryptedSecretInput);
                if (context) {
                    setOrigin(context);
                    setSetName(context.set.metadata.name);
                    setDraft(context.set);
                    setText(YAML.stringify(context.revision.spec.values));
                    setRestoredFrom(context.revision.metadata.name);
                    setHistory(historyPage.items || []);
                    setCursor(historyPage.metadata?.continue || '');
                    setDiff(comparison);
                    setLeft(context.revision.metadata.name);
                    setRight(context.set.spec.revision);
                }
            } catch (e) {
                if (current === scope.current) setError(e.response?.body?.message || e.message || String(e));
            } finally {
                if (current === scope.current) setBusy(false);
            }
        })();
        return () => {
            scope.current++;
        };
    }, [ns, requestedBinding, sourceWorkflow, sourceUID]);
    const edit = (next: string) => {
        setText(next);
        setPreview(undefined);
        setRetryPlan(undefined);
        setSubmitKey('');
    };
    const choose = async (name: string) => {
        const current = ++scope.current;
        setModel(undefined);
        setForm({});
        setFormValid(true);
        const chosen = bindings.find(x => x.metadata.name === name);
        setBinding(chosen);
        setDraft(undefined);
        setHistory([]);
        setPreview(undefined);
        setRetryPlan(undefined);
        setDiff(undefined);
        setWorkflow('');
        setRestoredFrom('');
        edit('{}\n');
        if (!chosen) return;
        const nextModel = (await request(ns, 'models/' + chosen.spec.parameterTemplate)).spec;
        const form = chosen.spec.form ? (await request(ns, 'forms/' + chosen.spec.form)).spec : {};
        if (current !== scope.current) return;
        setModel(nextModel);
        setForm(form);
    };
    const loadHistory = async (more = false) => {
        const current = scope.current;
        const page = await request(ns, 'history/' + encodeURIComponent(setName) + '?view=summary' + (more ? '&continue=' + encodeURIComponent(cursor) : ''));
        if (current !== scope.current) return;
        setHistory(old => (more ? [...old, ...(page.items || [])] : page.items || []));
        setCursor(page.metadata?.continue || '');
    };
    const compareOrigin = async (revision: string) => {
        if (!origin) return;
        const executed = origin.revision.metadata.name;
        setDiff(await request(ns, 'diff', 'post', {left: executed, right: revision}));
        setLeft(executed);
        setRight(revision);
    };
    let value: any;
    let parseError = '';
    try {
        value = YAML.parse(text, {uniqueKeys: true, maxAliasCount: 30});
    } catch (e) {
        parseError = e.message;
    }
    return (
        <Page title='Structured parameters'>
            <main className='parameterization'>
                <h2>
                    Structured parameters <small>Experimental</small>
                </h2>
                <div className='parameter-toolbar' aria-label='Parameter views'>
                    <button aria-pressed={view === 'inputs'} onClick={() => setView('inputs')}>
                        Input editor
                    </button>
                    <button aria-pressed={view === 'resources'} onClick={() => setView('resources')}>
                        Resource definitions and records
                    </button>
                </div>
                {view === 'resources' ? (
                    <ResourceCatalog namespace={ns} />
                ) : (
                    <>
                        <p>Save reusable input revisions, compare changes, and submit a native Argo Workflow.</p>
                        <p>Draft saves do not start a Workflow. Parameter revisions retain Secret references; new secret values use separate encrypted storage.</p>
                        {origin && (
                            <p role='status'>
                                Loaded execution revision <code>{origin.revision.metadata.name}</code> from <code>{origin.source.workflow}</code>. Current saved revision:{' '}
                                <code>{draft?.spec.revision}</code>. Editing uses the current binding and form; review the differences before saving. Preview partial retry to keep
                                this Workflow and reuse explicitly permitted results. Submitting as a new Workflow is a separate action.
                            </p>
                        )}
                        {error && (
                            <div role='alert' className='error'>
                                {error}
                            </div>
                        )}
                        <div className='parameter-toolbar'>
                            <label>
                                Workflow binding{' '}
                                <select
                                    aria-label='Workflow binding'
                                    disabled={busy || !!sourceWorkflow}
                                    value={binding?.metadata.name || ''}
                                    onChange={e => run(() => choose(e.target.value))}>
                                    <option value=''>Choose a binding</option>
                                    {bindings.map(x => (
                                        <option key={x.metadata.name}>{x.metadata.name}</option>
                                    ))}
                                </select>
                            </label>
                            <label>
                                Parameter set{' '}
                                <input
                                    aria-label='Parameter set'
                                    disabled={busy || !!sourceWorkflow}
                                    value={setName}
                                    onChange={e => {
                                        setSetName(e.target.value);
                                        setDraft(undefined);
                                        setPreview(undefined);
                                        setRetryPlan(undefined);
                                        setHistory([]);
                                    }}
                                />
                            </label>
                            <button
                                disabled={busy || !binding}
                                onClick={() =>
                                    run(async () => {
                                        const data = await request(ns, 'sets/' + setName);
                                        if (data.spec.template !== binding.spec.parameterTemplate) throw new Error('This set uses a different parameter model.');
                                        setDraft(data);
                                        edit(YAML.stringify(data.spec.values));
                                        setRestoredFrom('');
                                        await compareOrigin(data.spec.revision);
                                        await loadHistory();
                                    })
                                }>
                                Load saved set
                            </button>
                        </div>
                        {model && (
                            <>
                                <div className='parameter-toolbar'>
                                    <button onClick={() => setMode('form')}>Form</button>
                                    <button onClick={() => setMode('yaml')}>YAML</button>
                                </div>
                                {parseError && <div role='alert'>{parseError}</div>}
                                {mode === 'form' && !parseError && (
                                    <DeclaredForm
                                        key={binding.metadata.name}
                                        schema={model.schema}
                                        value={value}
                                        onChange={v => edit(YAML.stringify(v))}
                                        form={form}
                                        onValidityChange={setFormValid}
                                        disabled={busy}
                                        optionContext={{namespace: ns, binding: binding.metadata.name, values: value, sources: {}}}
                                        secretContext={{
                                            namespace: ns,
                                            template: binding.spec.parameterTemplate,
                                            paths: model.sensitivePaths || [],
                                            enabled: secretInput,
                                            identityRules: model.identityRules || [],
                                            onBusyChange: setBusy
                                        }}
                                    />
                                )}
                                {mode === 'yaml' && (
                                    <SuspenseMonacoEditor
                                        height={500}
                                        language='yaml'
                                        value={text}
                                        options={{automaticLayout: true, folding: true, readOnly: busy, minimap: {enabled: false}}}
                                        onChange={edit}
                                    />
                                )}
                                <label>
                                    Change note <input aria-label='Change note' value={message} onChange={e => setMessage(e.target.value)} />
                                </label>
                                <div className='parameter-toolbar'>
                                    <button
                                        disabled={busy || !!parseError || (mode === 'form' && !formValid)}
                                        onClick={() =>
                                            run(async () => {
                                                const data = await request(ns, 'sets/' + setName, 'put', {
                                                    template: binding.spec.parameterTemplate,
                                                    values: value,
                                                    resourceVersion: draft?.metadata.resourceVersion || '',
                                                    message,
                                                    restoredFrom
                                                });
                                                setDraft(data);
                                                edit(YAML.stringify(data.spec.values));
                                                setRestoredFrom('');
                                                await compareOrigin(data.spec.revision);
                                                await loadHistory();
                                            })
                                        }>
                                        Save revision
                                    </button>
                                    <button
                                        disabled={busy || !draft || (mode === 'form' && !formValid) || text !== YAML.stringify(draft.spec.values)}
                                        onClick={() =>
                                            run(async () => {
                                                setPreview(await request(ns, 'preview', 'post', {binding: binding.metadata.name, revision: draft.spec.revision}));
                                                setSubmitKey(crypto.randomUUID());
                                            })
                                        }>
                                        Validate and preview
                                    </button>
                                </div>
                                {draft && (
                                    <p>
                                        Saved revision: <code>{draft.spec.revision}</code>
                                    </p>
                                )}
                                {history.length > 0 && (
                                    <details open={true}>
                                        <summary>Revision history ({history.length})</summary>
                                        <button disabled={busy} onClick={() => run(() => loadHistory())}>
                                            Reload history
                                        </button>
                                        <table>
                                            <thead>
                                                <tr>
                                                    <th>Version</th>
                                                    <th>Saved by / time</th>
                                                    <th>Note</th>
                                                    <th>Actions</th>
                                                </tr>
                                            </thead>
                                            <tbody>
                                                {history.map(x => (
                                                    <tr key={x.metadata.name}>
                                                        <td>
                                                            <code>{x.metadata.name}</code>
                                                        </td>
                                                        <td>
                                                            {x.spec.actor}
                                                            <br />
                                                            {x.spec.savedAt}
                                                        </td>
                                                        <td>{x.spec.message}</td>
                                                        <td>
                                                            <button
                                                                disabled={busy}
                                                                onClick={() =>
                                                                    run(async () => {
                                                                        const current = scope.current;
                                                                        const revision = await request(ns, 'revisions/' + encodeURIComponent(x.metadata.name));
                                                                        if (current !== scope.current) return;
                                                                        if (revision.spec.set !== setName || (x.metadata.uid && revision.metadata.uid !== x.metadata.uid))
                                                                            throw new Error('Revision identity changed; refresh history.');
                                                                        edit(YAML.stringify(revision.spec.values));
                                                                        setRestoredFrom(x.metadata.name);
                                                                    })
                                                                }>
                                                                Restore into editor
                                                            </button>
                                                            <button onClick={() => setLeft(x.metadata.name)}>Compare from</button>
                                                            <button onClick={() => setRight(x.metadata.name)}>Compare to</button>
                                                        </td>
                                                    </tr>
                                                ))}
                                            </tbody>
                                        </table>
                                        {cursor && <button onClick={() => run(() => loadHistory(true))}>Load older revisions</button>}
                                        <p>
                                            Compare {left} → {right}
                                        </p>
                                        <button disabled={!left || !right || busy} onClick={() => run(async () => setDiff(await request(ns, 'diff', 'post', {left, right})))}>
                                            Compare revisions
                                        </button>
                                        {diff && <PagedRevisionDiff namespace={ns} comparison={diff} />}
                                    </details>
                                )}
                                {origin && draft?.spec.revision && (
                                    <section aria-label='Partial retry'>
                                        <h3>Retry with this saved revision</h3>
                                        <p>Preview checks affected task inputs and result reuse. Unsupported changes do not create another Workflow.</p>
                                        <button
                                            disabled={busy}
                                            onClick={() =>
                                                run(async () => {
                                                    const plan = await request(ns, 'retry-preview', 'post', {...origin.source, revision: draft.spec.revision});
                                                    setRetryPlan(plan);
                                                    setSubmitKey('retry-' + Date.now() + '-' + Math.random().toString(36).slice(2));
                                                })
                                            }>
                                            Preview partial retry
                                        </button>
                                        {retryPlan && (
                                            <>
                                                <p>
                                                    Affected tasks: {retryPlan.affectedTasks.join(', ')}. Retained results and approvals:{' '}
                                                    {Object.keys(retryPlan.retainedNodes).length}.
                                                </p>
                                                <pre aria-label='Partial retry plan'>{JSON.stringify(retryPlan, null, 2)}</pre>
                                                <button
                                                    disabled={busy}
                                                    onClick={() =>
                                                        run(async () => {
                                                            const wf = await request(ns, 'retry', 'post', {
                                                                ...origin.source,
                                                                revision: retryPlan.revision,
                                                                resourceVersion: retryPlan.resourceVersion,
                                                                planDigest: retryPlan.digest,
                                                                idempotencyKey: submitKey
                                                            });
                                                            setWorkflow(wf.metadata.name);
                                                            setRetryPlan(undefined);
                                                        })
                                                    }>
                                                    Apply revision and retry this Workflow
                                                </button>
                                            </>
                                        )}
                                    </section>
                                )}
                                {preview && (
                                    <details open={true}>
                                        <summary>Execution preview</summary>
                                        <p>
                                            Digest: <code>{preview.executionDigest}</code>
                                        </p>
                                        <pre>{YAML.stringify(preview.workflow || preview.errors)}</pre>
                                        <button
                                            disabled={busy || !!preview.errors?.length || !preview.workflow}
                                            onClick={() =>
                                                run(async () => {
                                                    const wf = await request(ns, 'submit', 'post', {
                                                        binding: binding.metadata.name,
                                                        revision: draft.spec.revision,
                                                        executionDigest: preview.executionDigest,
                                                        idempotencyKey: submitKey,
                                                        ...(origin ? {sourceExecution: origin.source} : {})
                                                    });
                                                    setWorkflow(wf.metadata.name);
                                                })
                                            }>
                                            {origin ? 'Submit as a new Workflow' : 'Submit this revision'}
                                        </button>
                                    </details>
                                )}
                                {workflow && (
                                    <p>
                                        Workflow: <a href={uiUrl('workflows/' + ns + '/' + workflow)}>{workflow}</a>
                                    </p>
                                )}
                            </>
                        )}
                    </>
                )}
            </main>
        </Page>
    );
}

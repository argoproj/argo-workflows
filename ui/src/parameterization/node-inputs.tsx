import * as React from 'react';
import {useEffect, useState} from 'react';

import {Workflow} from '../shared/models';
import {request} from './api';
import {DeclaredForm} from './declared-form';

import './parameterization.scss';

export function WaitingInput({workflow, item, encryptedSecretInput = false}: {workflow: Workflow; item: any; encryptedSecretInput?: boolean}) {
    const [value, setValue] = useState<any>({});
    const [busy, setBusy] = useState(false);
    const [secretBusy, setSecretBusy] = useState(false);
    const [error, setError] = useState('');
    const [formValid, setFormValid] = useState(true);
    const [submitted, setSubmitted] = useState(false);
    // Retain the exact request after an uncertain response, so retrying cannot
    // silently replace a committed input or resume another node.
    const [attempt, setAttempt] = useState<any>();
    if (submitted) return <p>Input accepted for {item.name}.</p>;
    return (
        <section className='white-box'>
            <h4>{item.name}: required input</h4>
            <DeclaredForm
                schema={item.contract.model.schema}
                value={value}
                form={item.contract.form || {}}
                onValidityChange={setFormValid}
                disabled={busy || secretBusy || !!attempt}
                onChange={setValue}
                optionContext={{
                    namespace: workflow.metadata.namespace,
                    binding: '',
                    sources: {},
                    values: value,
                    input: {workflow: workflow.metadata.name, uid: workflow.metadata.uid, nodeId: item.nodeId, contractDigest: item.contractDigest}
                }}
                secretContext={{
                    namespace: workflow.metadata.namespace,
                    template: '',
                    paths: item.contract.model.sensitivePaths || [],
                    enabled: encryptedSecretInput,
                    identityRules: item.contract.model.identityRules || [],
                    onBusyChange: setSecretBusy,
                    input: {workflow: workflow.metadata.name, uid: workflow.metadata.uid, nodeId: item.nodeId, contractDigest: item.contractDigest}
                }}
            />
            {error && <p role='alert'>{error}</p>}
            <button
                className='argo-button'
                disabled={busy || secretBusy || !formValid}
                onClick={async () => {
                    setBusy(true);
                    setError('');
                    try {
                        let next = attempt;
                        if (!next) {
                            const latest = await request(workflow.metadata.namespace, 'inputs/' + workflow.metadata.name);
                            if (latest.uid !== workflow.metadata.uid || !latest.items.some((x: any) => x.nodeId === item.nodeId && x.contractDigest === item.contractDigest))
                                throw new Error('This workflow or input contract changed; reopen the pending input.');
                            next = {uid: latest.uid, resourceVersion: latest.resourceVersion, nodeId: item.nodeId, requestId: crypto.randomUUID(), values: value};
                            setAttempt(next);
                        }
                        await request(workflow.metadata.namespace, 'inputs/' + workflow.metadata.name, 'post', next);
                        setSubmitted(true);
                    } catch (e) {
                        setError(e.response?.body?.message || e.message);
                        // Confirmed validation/CAS rejection has not resumed this node.
                        if (e.status === 400 || e.status === 409) setAttempt(undefined);
                    } finally {
                        setBusy(false);
                    }
                }}>
                {attempt ? 'Recover input submission' : 'Validate input and continue this node'}
            </button>
        </section>
    );
}

export function NodeInputs({workflow}: {workflow: Workflow}) {
    const [items, setItems] = useState<any[]>([]);
    const [encryptedSecretInput, setEncryptedSecretInput] = useState(false);
    const [error, setError] = useState('');
    const [submitted, setSubmitted] = useState<any[]>([]);
    useEffect(() => {
        let active = true;
        request(workflow.metadata.namespace, 'inputs/' + workflow.metadata.name)
            .then(data => {
                if (active) {
                    setItems(data.items || []);
                    setEncryptedSecretInput(!!data.encryptedSecretInput);
                    setSubmitted(data.submitted || []);
                    setError('');
                }
            })
            .catch(e => {
                if (active) setError(e.response?.body?.message || e.message);
            });
        return () => {
            active = false;
        };
    }, [workflow.metadata.namespace, workflow.metadata.name, workflow.metadata.resourceVersion]);
    if (!error && !items.length && !submitted.length) return null;
    return (
        <div className='parameterization' style={{margin: 16}}>
            {error && <p role='alert'>{error}</p>}
            {items.map(item => (
                <WaitingInput
                    key={workflow.metadata.uid + ':' + item.nodeId + ':' + item.contractDigest}
                    workflow={workflow}
                    item={item}
                    encryptedSecretInput={encryptedSecretInput}
                />
            ))}
            {submitted.length > 0 && (
                <details>
                    <summary>Submitted node inputs ({submitted.length})</summary>
                    {submitted.map(record => (
                        <section key={record.nodeId}>
                            <p>
                                {record.nodeId} · {record.actor} · {record.submittedAt}
                            </p>
                            <pre>{JSON.stringify(record.values, null, 2)}</pre>
                        </section>
                    ))}
                </details>
            )}
        </div>
    );
}

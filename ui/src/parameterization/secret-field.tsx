import * as React from 'react';
import {useEffect, useRef, useState} from 'react';

import {request} from './api';

export interface SecretContext {
    namespace: string;
    template: string;
    input?: {workflow: string; uid: string; nodeId: string; contractDigest: string};
    paths: string[];
    enabled: boolean;
    rowIdentity?: string;
    identityRules?: {path: string; keyPath: string}[];
    onBusyChange?: (busy: boolean) => void;
}

// New plaintext lives only in this mounted widget until the dedicated write.
// Ordinary form/YAML/history state receives the resulting reference only.
export function SecretField({context, path, value, onChange, disabled}: {context: SecretContext; path: string; value: any; onChange: (v: any) => void; disabled: boolean}) {
    const [input, setInput] = useState('');
    const [requestId, setRequestId] = useState('');
    const [busy, setBusy] = useState(false);
    const [error, setError] = useState('');
    const current = useRef(0);
    const inputIdentity = JSON.stringify(context.input || {});
    const latestChange = useRef(onChange);
    latestChange.current = onChange;
    useEffect(() => {
        current.current++;
        setInput('');
        setRequestId('');
        setBusy(false);
        return () => {
            current.current++;
        };
    }, [context.namespace, context.template, inputIdentity, path, value?.secretRef?.name, value?.secretRef?.key, value?.latestSecretRef?.configMap, value?.latestSecretRef?.key]);
    return (
        <div className='parameter-field'>
            <strong>{path}</strong>
            <span>
                {value?.secretRef
                    ? `Secret reference: ${value.secretRef.name} / ${value.secretRef.key}`
                    : value?.latestSecretRef
                      ? `Latest at submission: ${value.latestSecretRef.configMap} / ${value.latestSecretRef.key}`
                      : 'No secret reference'}
            </span>
            <label>
                Version policy
                <select
                    aria-label={'Secret version policy ' + path}
                    disabled={disabled || busy}
                    value={value?.latestSecretRef ? 'latest' : 'fixed'}
                    onChange={e => onChange(e.target.value === 'latest' ? {latestSecretRef: {configMap: '', key: 'current'}} : undefined)}>
                    <option value='fixed'>Fixed immutable version</option>
                    <option value='latest'>Latest at submission</option>
                </select>
            </label>
            {value?.latestSecretRef && (
                <>
                    <label>
                        Version index ConfigMap
                        <input
                            aria-label={'Secret version index ' + path}
                            disabled={disabled || busy}
                            value={value.latestSecretRef.configMap || ''}
                            onChange={e => onChange({latestSecretRef: {...value.latestSecretRef, configMap: e.target.value}})}
                        />
                    </label>
                    <label>
                        Index key
                        <input
                            aria-label={'Secret version index key ' + path}
                            disabled={disabled || busy}
                            value={value.latestSecretRef.key || ''}
                            onChange={e => onChange({latestSecretRef: {...value.latestSecretRef, key: e.target.value}})}
                        />
                    </label>
                    <small>
                        The model must allow latestSecretRef. Preview resolves an authorized immutable version. Original-parameter retries keep that version; reviewed parameter
                        retries can select a newer version.
                    </small>
                </>
            )}
            {context.enabled ? (
                <>
                    <input
                        aria-label={'New secret ' + path}
                        type='password'
                        autoComplete='new-password'
                        disabled={disabled || busy}
                        value={input}
                        onChange={e => {
                            setInput(e.target.value);
                            setRequestId(crypto.randomUUID());
                            setError('');
                        }}
                    />
                    <button
                        disabled={disabled || busy || !requestId}
                        onClick={async () => {
                            const generation = current.current;
                            setBusy(true);
                            context.onBusyChange?.(true);
                            setError('');
                            try {
                                const ref = await request(context.namespace, 'secrets', 'post', {
                                    ...(context.input ? {input: context.input} : {template: context.template}),
                                    path,
                                    requestId,
                                    value: input
                                });
                                if (generation !== current.current) return;
                                setInput('');
                                setRequestId('');
                                latestChange.current(ref);
                            } catch {
                                // Do not forward arbitrary provider/request errors containing plaintext.
                                if (generation !== current.current) return;
                                setError('Secret storage failed. Verify authorization and backend availability, then retry.');
                            } finally {
                                if (generation === current.current) setBusy(false);
                                context.onBusyChange?.(false);
                            }
                        }}>
                        Store new encrypted secret version
                    </button>
                </>
            ) : (
                <small>Use an immutable Secret reference in YAML. Encrypted input is not configured on this server.</small>
            )}
            <button disabled={disabled || busy} onClick={() => onChange(undefined)}>
                Clear reference
            </button>
            {error && <span role='alert'>{error}</span>}
        </div>
    );
}

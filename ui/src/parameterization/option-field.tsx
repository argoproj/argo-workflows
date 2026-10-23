import React, {useEffect, useState} from 'react';

import {request} from './api';
import {at} from './form-definition';

export interface OptionContext {
    namespace: string;
    binding: string;
    input?: {workflow: string; uid: string; nodeId: string; contractDigest: string};
    values: any;
    sources: {[path: string]: {allowManual?: boolean; dependsOn?: string[]}};
}
export function OptionField({
    context,
    path,
    sourceKey = path,
    value,
    onChange,
    disabled
}: {
    sourceKey?: string;
    context: OptionContext;
    path: string;
    value: any;
    onChange: (value: any) => void;
    disabled: boolean;
}) {
    const [result, setResult] = useState<any>();
    const [error, setError] = useState('');
    const [loading, setLoading] = useState(false);
    const body = JSON.stringify({...(context.input ? {input: context.input} : {binding: context.binding}), path, values: context.values});
    const source = context.sources[sourceKey];
    const sourceIdentity = JSON.stringify(source || {});
    const missingDependencies = (source?.dependsOn || []).filter(dependency => !at(context.values, dependency).exists);
    useEffect(() => {
        let active = true;
        setLoading(true);
        setResult(undefined);
        setError('');
        if (missingDependencies.length > 0) {
            setLoading(false);
            return () => {
                active = false;
            };
        }
        request(context.namespace, 'options', 'post', JSON.parse(body))
            .then(data => {
                if (active) setResult(data);
            })
            .catch(e => {
                if (active) setError(e.response?.body?.message || e.message);
            })
            .finally(() => {
                if (active) setLoading(false);
            });
        return () => {
            active = false;
        };
    }, [context.namespace, body, sourceIdentity]);
    const selected = JSON.stringify(value) || '';
    const invalid = result && value !== undefined && !result.items.some((item: any) => JSON.stringify(item.value) === selected && !item.disabled);
    return (
        <span>
            <select
                aria-label={path}
                disabled={disabled || loading || !!error || missingDependencies.length > 0}
                value={selected}
                onChange={e => onChange(e.target.value === '' ? undefined : JSON.parse(e.target.value))}>
                <option value=''>Choose a value</option>
                {missingDependencies.length > 0 && value !== undefined && (
                    <option value={selected} disabled={true}>
                        Saved selection (waiting for dependencies)
                    </option>
                )}
                {invalid && (
                    <option value={selected} disabled={true}>
                        Previous selection is unavailable
                    </option>
                )}
                {(result?.items || []).map((item: any) => (
                    <option key={JSON.stringify(item.value)} value={JSON.stringify(item.value)} disabled={item.disabled}>
                        {item.label}
                    </option>
                ))}
            </select>
            {missingDependencies.length > 0 && <small>Choose dependent values first: {missingDependencies.join(', ')}</small>}
            {loading && <small>Loading options…</small>}
            {error && <small role='alert'>{error}</small>}
            {result && result.items.length === 0 && <small>No matching options.</small>}
            {context.sources[sourceKey]?.allowManual && (
                <input
                    aria-label={path + ' manual'}
                    disabled={disabled || missingDependencies.length > 0}
                    value={value === undefined ? '' : String(value)}
                    onChange={e => onChange(e.target.value)}
                />
            )}
        </span>
    );
}

import * as React from 'react';
import {useEffect, useState} from 'react';

import {at, effectiveOptions, FormDefinition, LayoutNode, schemaAt, setAt, validateForm} from './form-definition';
import {OptionContext} from './option-field';
import {SchemaForm} from './schema-form';
import {SecretContext} from './secret-field';

function LayoutChildren({parent = '', nodes, render}: {parent?: string; nodes: LayoutNode[]; render: (node: LayoutNode) => React.ReactNode}) {
    const children = nodes.filter(n => (n.parent || '') === parent);
    const tabs = children.filter(n => n.kind === 'tab');
    const [selected, select] = useState('');
    const active = tabs.find(n => n.id === selected) || tabs[0];
    return (
        <>
            {tabs.length > 0 && (
                <>
                    <div role='tablist' aria-label={parent || 'Form sections'}>
                        {tabs.map((tab, index) => (
                            <button
                                key={tab.id}
                                type='button'
                                role='tab'
                                aria-selected={active === tab}
                                tabIndex={active === tab ? 0 : -1}
                                onClick={() => select(tab.id)}
                                onKeyDown={e => {
                                    const next =
                                        e.key === 'ArrowRight'
                                            ? (index + 1) % tabs.length
                                            : e.key === 'ArrowLeft'
                                              ? (index + tabs.length - 1) % tabs.length
                                              : e.key === 'Home'
                                                ? 0
                                                : e.key === 'End'
                                                  ? tabs.length - 1
                                                  : -1;
                                    if (next < 0) return;
                                    e.preventDefault();
                                    select(tabs[next].id);
                                    (e.currentTarget.parentElement.children[next] as HTMLElement).focus();
                                }}>
                                {tab.label || tab.id}
                            </button>
                        ))}
                    </div>
                    {tabs.map(tab => (
                        <section key={tab.id} hidden={active !== tab} role='tabpanel' aria-label={tab.label || tab.id}>
                            {render(tab)}
                        </section>
                    ))}
                </>
            )}
            {children
                .filter(n => n.kind !== 'tab')
                .map(node =>
                    node.kind === 'tree' ? (
                        <details key={node.id} open={!node.collapsed} className='parameter-group'>
                            <summary>{node.label || node.id}</summary>
                            {render(node)}
                        </details>
                    ) : (
                        <fieldset key={node.id} className='parameter-group'>
                            <legend>{node.label || node.id}</legend>
                            {render(node)}
                        </fieldset>
                    )
                )}
        </>
    );
}

export function DeclaredForm({
    schema,
    value,
    onChange,
    form = {},
    disabled = false,
    optionContext,
    secretContext,
    onValidityChange
}: {
    schema: any;
    value: any;
    onChange: (v: any) => void;
    form?: FormDefinition;
    disabled?: boolean;
    optionContext?: OptionContext;
    secretContext?: SecretContext;
    onValidityChange?: (valid: boolean) => void;
}) {
    const problems = validateForm(form, schema);
    for (const field of form.fields || [])
        if (field.widget === 'secret' && !secretContext?.paths.includes(field.path)) problems.push('Secret widget requires a declared sensitive parameter: ' + field.path);
    const [editorErrors, setEditorErrors] = useState<{[path: string]: string}>({});
    const reportError = React.useCallback(
        (path: string, message: string) =>
            setEditorErrors(old => {
                if ((old[path] || '') === message) return old;
                const next = {...old};
                if (message) next[path] = message;
                else delete next[path];
                return next;
            }),
        []
    );
    const valid = !problems.length && !Object.keys(editorErrors).length;
    useEffect(() => {
        onValidityChange?.(valid);
    }, [valid, onValidityChange]);
    if (problems.length) return <div role='alert'>Invalid form definition: {problems.join('; ')}</div>;
    const fields = (form.fields || []).filter(f => f.group);
    const context = {...optionContext, sources: effectiveOptions(form)};
    const common = {
        root: schema,
        labels: form.labels || {},
        disabled,
        optionContext: optionContext ? context : undefined,
        secretContext,
        formContext: {definition: form, values: value, reportError}
    };
    const render = (node: LayoutNode): React.ReactNode => (
        <>
            {fields
                .filter(f => f.group === node.id)
                .map(field => (
                    <SchemaForm
                        key={field.path}
                        {...common}
                        schema={schemaAt(schema, field.path)}
                        path={field.path}
                        value={at(value, field.path).value}
                        onChange={v => onChange(setAt(value, field.path, v))}
                    />
                ))}
            <LayoutChildren parent={node.id} nodes={form.layout || []} render={render} />
        </>
    );
    return (
        <>
            <LayoutChildren nodes={form.layout || []} render={render} />
            <SchemaForm {...common} schema={schema} value={value} excludedPaths={fields.map(f => f.path)} onChange={onChange} />
            {Object.entries(editorErrors).map(([path, error]) => (
                <p key={path} role='alert'>
                    {path}: {error}
                </p>
            ))}
        </>
    );
}

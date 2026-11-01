import * as React from 'react';
import * as YAML from 'yaml';

import {at, fieldFor, FormDefinition, predicateMatches, schemaAt} from './form-definition';
import {OptionContext, OptionField} from './option-field';
import {SecretContext, SecretField} from './secret-field';

export const pointerKey = (key: string) => key.replace(/~/g, '~0').replace(/\//g, '~1');
export const initialValue = (schema: any): any => {
    if (Object.prototype.hasOwnProperty.call(schema, 'default')) return JSON.parse(JSON.stringify(schema.default));
    if (schema.type === 'object') return {};
    if (schema.type === 'array') return [];
    if (schema.type === 'boolean') return false;
    if (schema.type === 'number' || schema.type === 'integer') return 0;
    return '';
};
interface FormContext {
    definition: FormDefinition;
    values: any;
    reportError: (path: string, message: string) => void;
}
function YAMLField({
    path,
    value,
    onChange,
    disabled,
    reportError
}: {
    path: string;
    value: any;
    onChange: (v: any) => void;
    disabled: boolean;
    reportError?: FormContext['reportError'];
}) {
    const serialized = YAML.stringify(value) || '';
    const [text, setText] = React.useState(serialized);
    React.useEffect(() => {
        setText(serialized);
        reportError?.(path, '');
    }, [serialized, path, reportError]);
    React.useEffect(() => () => reportError?.(path, ''), [path, reportError]);
    return (
        <textarea
            aria-label={path}
            disabled={disabled}
            value={text}
            onChange={e => {
                const next = e.target.value;
                setText(next);
                try {
                    const parsed = YAML.parse(next, {uniqueKeys: true, maxAliasCount: 30});
                    reportError?.(path, '');
                    onChange(parsed);
                } catch (_) {
                    reportError?.(path, 'Invalid YAML; correct this field before saving.');
                }
            }}
        />
    );
}

// Presentation never mutates hidden values. The server validates the full document.
export function SchemaForm({
    schema,
    root,
    value,
    onChange,
    path = '',
    labels = {},
    disabled = false,
    optionContext,
    secretContext,
    formContext,
    excludedPaths = []
}: {
    schema: any;
    root?: any;
    value: any;
    onChange: (v: any) => void;
    path?: string;
    labels?: {[key: string]: string};
    disabled?: boolean;
    optionContext?: OptionContext;
    secretContext?: SecretContext;
    formContext?: FormContext;
    excludedPaths?: string[];
}) {
    root = root || schema;
    schema = schemaAt(root, path) || schema;
    if (!schema || typeof schema !== 'object') return <span>Use YAML for this schema.</span>;
    if (excludedPaths.includes(path)) return null;
    const field = formContext && fieldFor(formContext.definition, path);
    const prefixes = [
        '',
        ...path
            .split('/')
            .slice(1)
            .map((_, index) =>
                path
                    .split('/')
                    .slice(0, index + 2)
                    .join('/')
            )
    ];
    for (const prefix of prefixes) {
        const policy = formContext && fieldFor(formContext.definition, prefix);
        if (policy && !predicateMatches(policy.visibleWhen, formContext.values, policy, prefix)) return null;
        disabled =
            disabled || schemaAt(root, prefix)?.readOnly === true || !!policy?.readOnly || (policy && !predicateMatches(policy.enabledWhen, formContext.values, policy, prefix));
    }
    disabled = !!disabled || schema.readOnly === true;
    const title = field?.label || labels[path] || schema.title || path.split('/').pop() || 'Parameters';
    const widget = field?.widget;
    const description = field?.description || schema.description;
    const childProps = {root, labels, disabled, optionContext, secretContext, formContext, excludedPaths};
    if (secretContext && fieldFor({fields: secretContext.paths.map(p => ({path: p}))}, path))
        return (
            <div className='parameter-field'>
                <span>{title}</span>
                <SecretField key={secretContext.rowIdentity} context={secretContext} path={path} value={value} onChange={onChange} disabled={disabled} />
            </div>
        );
    if (widget === 'secret') return <p role='alert'>Secret widget requires a declared sensitive parameter: {path}</p>;
    if (widget === 'yaml')
        return (
            <div className='parameter-field'>
                <span>{title}</span>
                <YAMLField path={path} value={value} disabled={disabled} onChange={onChange} reportError={formContext?.reportError} />
                {description && <small>{description}</small>}
            </div>
        );
    if (schema.type === 'object' || schema.properties) {
        const obj = value && typeof value === 'object' && !Array.isArray(value) ? value : {};
        const keys = Object.keys(schema.properties || {}).filter(key => !excludedPaths.includes(path + '/' + pointerKey(key)));
        if (!keys.length && schema.additionalProperties === false) return null;
        return (
            <details open={true} className='parameter-group'>
                <summary>{title}</summary>
                {description && <small>{description}</small>}
                {keys.map(key => (
                    <SchemaForm
                        key={key}
                        {...childProps}
                        schema={schema.properties[key]}
                        value={obj[key]}
                        path={path + '/' + pointerKey(key)}
                        onChange={next => {
                            const updated = {...obj};
                            if (next === undefined) delete updated[key];
                            else Object.defineProperty(updated, key, {value: next, enumerable: true, configurable: true, writable: true});
                            onChange(updated);
                        }}
                    />
                ))}
                {schema.additionalProperties !== false && <small>Additional keys can be edited in YAML.</small>}
            </details>
        );
    }
    if (schema.type === 'array') {
        const list = Array.isArray(value) ? value : [];
        const itemSchema = schemaAt(root, path + '/*') || schema.items || {};
        const identityRule = secretContext?.identityRules?.find(rule => fieldFor({fields: [{path: rule.path}]}, path));
        // Reset only secret widgets when row identity changes; keep the ID
        // editor mounted so typing an identity does not lose keyboard focus.
        const rowContext = (item: any, index: number) =>
            secretContext && {
                ...secretContext,
                rowIdentity: (secretContext.rowIdentity || '') + '/' + index + ':' + JSON.stringify(identityRule ? at(item, identityRule.keyPath).value : null)
            };
        const controls = (index: number) => (
            <>
                <button
                    type='button'
                    disabled={disabled || (schema.minItems !== undefined && list.length <= schema.minItems)}
                    onClick={() => onChange(list.filter((_, j) => j !== index))}
                    aria-label={'Remove ' + path + '/' + index}>
                    Remove item
                </button>
                <button
                    type='button'
                    disabled={disabled || index === 0}
                    onClick={() => {
                        const next = list.slice();
                        [next[index - 1], next[index]] = [next[index], next[index - 1]];
                        onChange(next);
                    }}>
                    Move up
                </button>
            </>
        );
        return (
            <details open={true} className='parameter-group'>
                <summary>
                    {title} ({list.length})
                </summary>
                {description && <small>{description}</small>}
                {widget === 'table' ? (
                    <div className='parameter-table'>
                        <table aria-label={title}>
                            <thead>
                                <tr>
                                    {Object.keys(itemSchema.properties || {}).map(key => (
                                        <th key={key}>
                                            {fieldFor(formContext?.definition || {}, path + '/*/' + pointerKey(key))?.label || itemSchema.properties[key].title || key}
                                        </th>
                                    ))}
                                    <th>Actions</th>
                                </tr>
                            </thead>
                            <tbody>
                                {list.map((item, index) => (
                                    <tr key={index}>
                                        {Object.keys(itemSchema.properties || {}).map(key => (
                                            <td key={key}>
                                                <SchemaForm
                                                    {...childProps}
                                                    secretContext={rowContext(item, index)}
                                                    schema={itemSchema.properties[key]}
                                                    value={item?.[key]}
                                                    path={path + '/' + index + '/' + pointerKey(key)}
                                                    onChange={next => {
                                                        const updated = {...item};
                                                        if (next === undefined) delete updated[key];
                                                        else Object.defineProperty(updated, key, {value: next, enumerable: true, configurable: true, writable: true});
                                                        onChange(list.map((old, j) => (index === j ? updated : old)));
                                                    }}
                                                />
                                            </td>
                                        ))}
                                        <td>{controls(index)}</td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                ) : (
                    list.map((item, index) => (
                        <div key={index} className='parameter-item'>
                            <SchemaForm
                                {...childProps}
                                secretContext={rowContext(item, index)}
                                schema={itemSchema}
                                value={item}
                                path={path + '/' + index}
                                onChange={next => onChange(list.map((old, j) => (index === j ? next : old)))}
                            />
                            {controls(index)}
                        </div>
                    ))
                )}
                <button
                    type='button'
                    disabled={disabled || (schema.maxItems !== undefined && list.length >= schema.maxItems)}
                    onClick={() => onChange([...list, initialValue(itemSchema)])}>
                    Add {title} item
                </button>
            </details>
        );
    }
    const sourceKey = optionContext && fieldFor({fields: Object.keys(optionContext.sources).map(key => ({path: key}))}, path)?.path;
    const input = sourceKey ? (
        <OptionField context={optionContext} path={path} sourceKey={sourceKey} value={value} onChange={onChange} disabled={disabled} />
    ) : widget === 'checkbox' ? (
        <input aria-label={path} type='checkbox' disabled={disabled} checked={value === true} onChange={e => onChange(e.target.checked)} />
    ) : schema.enum ? (
        <select aria-label={path} disabled={disabled} value={JSON.stringify(value) || ''} onChange={e => onChange(e.target.value === '' ? undefined : JSON.parse(e.target.value))}>
            <option value=''>Unset</option>
            {schema.enum.map((option: any) => (
                <option key={JSON.stringify(option)} value={JSON.stringify(option)}>
                    {String(option)}
                </option>
            ))}
        </select>
    ) : schema.type === 'boolean' ? (
        <select
            aria-label={path}
            disabled={disabled}
            value={value === undefined ? '' : String(value)}
            onChange={e => onChange(e.target.value === '' ? undefined : e.target.value === 'true')}>
            <option value=''>Unset</option>
            <option value='true'>true</option>
            <option value='false'>false</option>
        </select>
    ) : (
        <input
            aria-label={path}
            disabled={disabled}
            type={schema.type === 'integer' || schema.type === 'number' ? 'number' : 'text'}
            value={value === undefined || value === null ? '' : String(value)}
            onChange={e => onChange(schema.type === 'integer' || schema.type === 'number' ? (e.target.value === '' ? undefined : Number(e.target.value)) : e.target.value)}
        />
    );
    return (
        <div className='parameter-field'>
            <span>{title}</span>
            {input}
            <button disabled={disabled} type='button' onClick={() => onChange(undefined)}>
                Unset
            </button>
            {Array.isArray(schema.type) && schema.type.includes('null') && (
                <button type='button' disabled={disabled} onClick={() => onChange(null)}>
                    Set null
                </button>
            )}
            {description && <small>{description}</small>}
            <small>{value === undefined ? 'Absent' : value === null ? 'Explicit null' : 'Explicit value'}</small>
        </div>
    );
}

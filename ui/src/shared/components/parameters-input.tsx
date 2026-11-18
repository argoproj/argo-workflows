import {Select} from 'argo-ui/src/components/select/select';
import React from 'react';

import {EnumFrom, Parameter, ParameterSource} from '../models';
import {services} from '../services';
import {Tooltip} from './tooltip';

export function getValueFromParameter(p: Parameter) {
    if (p.value === undefined) {
        return p.default;
    } else {
        return p.value;
    }
}

interface ParametersInputProps {
    namespace: string;
    /**
     * The stored object these parameters are defined on; the server reads enumFrom from it.
     */
    source: ParameterSource;
    parameters: Parameter[];
    /**
     * Workflow-level parameters that enumFrom references ({{workflow.parameters.x}}) resolve against.
     * Defaults to `parameters`; set it when these are template inputs.
     */
    workflowParameters?: Parameter[];
    onChange: (parameters: Parameter[]) => void;
}

// gRPC status code the server returns when a referenced parameter has no value yet.
const grpcFailedPrecondition = 9;

const parameterReference = /{{\s*workflow\.parameters\.([^}\s]+)\s*}}/g;

// Values of the other parameters that this enumFrom references, so cascading dropdowns
// only re-resolve when a parameter they depend on changes.
function referencedParameters(enumFrom: EnumFrom, parameters: Parameter[]): Record<string, string> {
    const text = `${enumFrom.resource?.namespace ?? ''} ${enumFrom.labelSelector ?? ''}`;
    const result: Record<string, string> = {};
    for (const [, name] of text.matchAll(parameterReference)) {
        const p = parameters.find(x => x.name === name);
        result[name] = (p && getValueFromParameter(p)) || '';
    }
    return result;
}

function TextInput({value, onChange}: {value?: string; onChange: (value: string) => void}) {
    return <textarea className='argo-field' value={value} onChange={e => onChange(e.target.value)} />;
}

function DynamicEnumSelect({
    namespace,
    source,
    enumFrom,
    parameter,
    parameters,
    onChange
}: {
    namespace: string;
    source: ParameterSource;
    enumFrom: EnumFrom;
    parameter: Parameter;
    parameters: Parameter[];
    onChange: (value: string) => void;
}) {
    const [options, setOptions] = React.useState<string[] | null>(null);
    const [truncated, setTruncated] = React.useState(false);
    const [error, setError] = React.useState<string | null>(null);
    // Set when a parameter this one depends on has no value yet; not an error.
    const [waiting, setWaiting] = React.useState<string | null>(null);
    const sourceKey = JSON.stringify(source);
    // The enumFrom is only read here to know which parameters it depends on; the server uses the stored copy.
    const dependencyKey = JSON.stringify(referencedParameters(enumFrom, parameters));

    React.useEffect(() => {
        let cancelled = false;
        setOptions(null);
        setError(null);
        setWaiting(null);
        // Debounce so typing into a parent parameter doesn't fire a request per keystroke.
        const timer = setTimeout(() => {
            services.workflows
                .resolveParameterEnum(namespace, source, parameter.name, JSON.parse(dependencyKey))
                .then(res => {
                    if (!cancelled) {
                        setOptions(res.values);
                        setTruncated(res.truncated);
                    }
                })
                .catch(e => {
                    if (cancelled) {
                        return;
                    }
                    const message = e?.response?.body?.message || e?.message || 'unknown error';
                    if (e?.response?.body?.code === grpcFailedPrecondition) {
                        setWaiting(message);
                    } else {
                        setError(message);
                    }
                });
        }, 300);
        return () => {
            cancelled = true;
            clearTimeout(timer);
        };
    }, [namespace, sourceKey, parameter.name, dependencyKey]);

    const value = getValueFromParameter(parameter);
    if (error !== null || waiting !== null || (options !== null && options.length === 0)) {
        const note = error !== null ? `Failed to load options: ${error}` : waiting !== null ? `Options not loaded yet: ${waiting}` : 'No options found';
        return (
            <div>
                <div style={{color: error !== null ? 'red' : undefined, fontSize: '12px', marginBottom: '4px'}}>{note}</div>
                <TextInput value={value} onChange={onChange} />
            </div>
        );
    }
    if (options === null) {
        return (
            <div>
                <i className='fa fa-circle-notch fa-spin' /> Loading options...
            </div>
        );
    }
    return (
        <div>
            <Select value={value ?? ''} options={options.map(v => ({value: v, title: v}))} onChange={e => onChange(e.value)} />
            {truncated && <div style={{fontSize: '12px', marginTop: '4px'}}>Showing the first {options.length} options only. Narrow the list with a label selector.</div>}
        </div>
    );
}

export function ParametersInput(props: ParametersInputProps) {
    function onParameterChange(parameter: Parameter, value: string) {
        const newParameters: Parameter[] = props.parameters.map(p => ({
            ...p,
            value: p.name === parameter.name ? value : getValueFromParameter(p)
        }));
        props.onChange(newParameters);
    }

    function renderInput(parameter: Parameter) {
        const onChange = (value: string) => onParameterChange(parameter, value);
        if (parameter.enumFrom) {
            return (
                <DynamicEnumSelect
                    namespace={props.namespace}
                    source={props.source}
                    enumFrom={parameter.enumFrom}
                    parameter={parameter}
                    parameters={props.workflowParameters ?? props.parameters}
                    onChange={onChange}
                />
            );
        }
        if (parameter.enum) {
            return <Select value={getValueFromParameter(parameter) ?? ''} options={parameter.enum.map(value => ({value, title: value}))} onChange={e => onChange(e.value)} />;
        }
        return <TextInput value={getValueFromParameter(parameter)} onChange={onChange} />;
    }

    return (
        <>
            {props.parameters.map((parameter, index) => (
                <div key={parameter.name + '_' + index} style={{marginBottom: 14}}>
                    <label>{parameter.name}</label>
                    {parameter.description && (
                        <Tooltip content={parameter.description}>
                            <i className='fa fa-question-circle' style={{marginLeft: 4}} />
                        </Tooltip>
                    )}
                    {renderInput(parameter)}
                </div>
            ))}
        </>
    );
}

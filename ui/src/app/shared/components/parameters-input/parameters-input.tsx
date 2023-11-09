import {Select, Tooltip} from 'argo-ui';
import React from 'react';
import {Parameter} from '../../../../models';
import {Utils} from '../../utils';

interface ParametersInputProps {
    parameters: Parameter[];
    onChange: (parameters: Parameter[]) => void;
}

export function ParametersInput(props: ParametersInputProps) {
    function onParameterChange(parameter: Parameter, value: string) {
        const newParameters: Parameter[] = props.parameters.map(p => ({
            name: p.name,
            value: p.name === parameter.name ? value : Utils.getValueFromParameter(p),
            enum: p.enum
        }));
        props.onChange(newParameters);
    }

    function displaySelectFieldForEnumValues(parameter: Parameter) {
        return (
            <Select
                key={parameter.name}
                value={Utils.getValueFromParameter(parameter)}
                options={parameter.enum.map(value => ({
                    value,
                    title: value
                }))}
                onChange={e => onParameterChange(parameter, e.value)}
            />
        );
    }

    function displayInputFieldForSingleValue(parameter: Parameter) {
        return <textarea className='argo-field' value={Utils.getValueFromParameter(parameter)} onChange={e => onParameterChange(parameter, e.target.value)} />;
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
                    {(parameter.enum && displaySelectFieldForEnumValues(parameter)) || displayInputFieldForSingleValue(parameter)}
                </div>
            ))}
        </>
    );
}

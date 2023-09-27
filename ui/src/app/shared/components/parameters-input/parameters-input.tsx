import {Select, Tooltip} from 'argo-ui';
import React, {useState} from 'react';
import {Parameter} from '../../../../models';
import {Utils} from '../../utils';

interface ParametersInputProps {
    parameters: Parameter[];
    onChange?: (parameters: Parameter[]) => void;
}

export function ParametersInput(props: ParametersInputProps) {
    const [parameters, setParameters] = useState<Parameter[]>(props.parameters || []);

    function displaySelectFieldForEnumValues(parameter: Parameter) {
        return (
            <Select
                key={parameter.name}
                value={Utils.getValueFromParameter(parameter)}
                options={parameter.enum.map(value => ({
                    value,
                    title: value
                }))}
                onChange={e => {
                    const newParameters: Parameter[] = parameters.map(p => ({
                        name: p.name,
                        value: p.name === parameter.name ? e.value : Utils.getValueFromParameter(p),
                        enum: p.enum
                    }));
                    setParameters(newParameters);
                    onParametersChange(newParameters);
                }}
            />
        );
    }

    function displayInputFieldForSingleValue(parameter: Parameter) {
        return (
            <textarea
                className='argo-field'
                value={Utils.getValueFromParameter(parameter)}
                onChange={e => {
                    const newParameters: Parameter[] = parameters.map(p => ({
                        name: p.name,
                        value: p.name === parameter.name ? e.target.value : Utils.getValueFromParameter(p),
                        enum: p.enum
                    }));
                    setParameters(newParameters);
                    onParametersChange(newParameters);
                }}
            />
        );
    }

    function onParametersChange(parameters: Parameter[]) {
        if (props.onChange) {
            props.onChange(parameters);
        }
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

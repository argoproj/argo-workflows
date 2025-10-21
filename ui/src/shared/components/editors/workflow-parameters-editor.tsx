import * as React from 'react';

import {Arguments, Parameter, WorkflowSpec} from '../../models';
import {KeyValueEditor} from './key-value-editor';

export function WorkflowParametersEditor<T extends WorkflowSpec>(props: {value: T; onChange: (value: T) => void; onError: (error: Error) => void}) {
    const originalParameters = props.value?.arguments?.parameters || [];
    const parameterKeyValues =
        props.value &&
        props.value.arguments &&
        props.value.arguments.parameters &&
        props.value.arguments.parameters
            .map(param => [param.name, param.value])
            .reduce(
                (obj, [key, val]) => {
                    obj[key] = val;
                    return obj;
                },
                {} as {[key: string]: string}
            );

    return (
        <>
            <div className='white-box'>
                <h5>Parameters</h5>
                <KeyValueEditor
                    keyValues={parameterKeyValues}
                    onChange={parameters => {
                        if (!props.value.arguments) {
                            props.value.arguments = {parameters: []} as Arguments;
                        }
                        props.value.arguments.parameters = Object.entries(parameters).map(([k, v]) => {
                            const originalParam = originalParameters.find(param => param.name == k);
                            const newParam: Parameter = {
                                name: k,
                                value: v
                            };

                            if (originalParam?.valueFrom) {
                                newParam.valueFrom = originalParam.valueFrom;
                            }
                            return newParam;
                        });
                        props.onChange(props.value);
                    }}
                />
            </div>
        </>
    );
}

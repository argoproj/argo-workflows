import * as React from 'react';
import { Parameter } from "../../../../models";
import {Select} from 'argo-ui';

interface SuspendInputProps {
    parameters: Parameter[]
    nodeId: string
    setParameter: (key: string, value: string) => void
}

export const SuspendInputs = (props: SuspendInputProps) => {

    const [parameters, setParameters] = React.useState(props.parameters)

    const setParameter = (key: string, value: string) => {
        props.setParameter(key, value);
        setParameters(previous => {
            return previous.map(parameter => {
                if (parameter.name === key) {
                    parameter.value = value;
                }
                return parameter;
            });
        })
    }

    const renderSelectField = (parameter: Parameter, index: Number) => {
        return (
            <React.Fragment key={parameter.name + "_" + index}>
                <br />
                <label>{parameter.name}</label>
                <Select
                    value={parameter.value || parameter.default}
                    options={parameter.enum.map(value => ({
                        value,
                        title: value
                    }))}
                    onChange={selected => {
                        setParameter(parameter.name, selected.value)
                    }}
                />
            </React.Fragment>
        );
    }

    const renderInputField = (parameter: Parameter, index: Number) => {
        return (
            <React.Fragment key={parameter.name + "_" + index}>
                <br />
                <label>{parameter.name}</label>
                <input
                    className='argo-field'
                    defaultValue={parameter.value || parameter.default}
                    onChange={event => {
                        setParameter(parameter.name, event.target.value)
                    }}
                />
            </React.Fragment>
        );
    }


    const renderFields = (parameter: Parameter, index: Number) => {
        if (parameter.enum) {
            return renderSelectField(parameter, index);
        }
        return renderInputField(parameter, index);
    }


    return (
        <div>
            <h2>Modify parameters</h2>
            {parameters.map((parameter, i) => renderFields(parameter, i))}
            <br />
            <br />
            Are you sure you want to resume node {props.nodeId} ?
        </div>
    );
}

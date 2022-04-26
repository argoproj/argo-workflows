import {Select} from 'argo-ui';
import * as React from 'react';
import {Parameter} from '../../../../models';

interface SuspendInputProps {
    parameters: Parameter[];
    nodeId: string;
    setParameter: (key: string, value: string) => void;
}

export const SuspendInputs = (props: SuspendInputProps) => {

    const renderSelectField = (parameter: Parameter) => {
        return (
            <React.Fragment key={parameter.name}>
                <br />
                <label>{parameter.name}</label>
                <Select
                    value={parameter.value || parameter.default}
                    options={parameter.enum.map(value => ({
                        value,
                        title: value
                    }))}
                    onChange={selected => {
                        props.setParameter(parameter.name, selected.value);
                    }}
                />
            </React.Fragment>
        );
    };

    const renderInputField = (parameter: Parameter) => {
        return (
            <React.Fragment key={parameter.name}>
                <br />
                <label>{parameter.name}</label>
                <input
                    className='argo-field'
                    defaultValue={parameter.value || parameter.default}
                    onChange={event => {
                        props.setParameter(parameter.name, event.target.value);
                    }}
                />
            </React.Fragment>
        );
    };

    const renderFields = (parameter: Parameter) => {
        if (parameter.enum) {
            return renderSelectField(parameter);
        }
        return renderInputField(parameter);
    };

    return (
        <div>
            <h2>Modify parameters</h2>
            {props.parameters.map(renderFields)}
            <br />
            <br />
            Are you sure you want to resume node {props.nodeId} ?
        </div>
    );
};

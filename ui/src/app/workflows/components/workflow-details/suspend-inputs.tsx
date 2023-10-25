import {Select, Tooltip} from 'argo-ui';
import * as React from 'react';
import {Parameter} from '../../../../models';

interface SuspendInputProps {
    parameters: Parameter[];
    nodeId: string;
    setParameter: (key: string, value: string) => void;
}

export function SuspendInputs(props: SuspendInputProps) {
    const [parameters, setParameters] = React.useState(props.parameters);

    const setParameter = (key: string, value: string) => {
        props.setParameter(key, value);
        setParameters(previous => {
            return previous.map(param => {
                if (param.name === key) {
                    param.value = value;
                }
                return param;
            });
        });
    };

    const renderSelectField = (parameter: Parameter) => {
        return (
            <React.Fragment key={parameter.name}>
                <br />
                <label>{parameter.name}</label>
                {parameter.description && (
                    <Tooltip content={parameter.description}>
                        <i className='fa fa-question-circle' style={{marginLeft: 4}} />
                    </Tooltip>
                )}
                <Select
                    value={parameter.value || parameter.default}
                    options={parameter.enum.map(value => ({
                        value,
                        title: value
                    }))}
                    onChange={selected => setParameter(parameter.name, selected.value)}
                />
            </React.Fragment>
        );
    };

    const renderInputField = (parameter: Parameter) => {
        return (
            <React.Fragment key={parameter.name}>
                <br />
                <label>{parameter.name}</label>
                <input className='argo-field' defaultValue={parameter.value || parameter.default} onChange={event => setParameter(parameter.name, event.target.value)} />
            </React.Fragment>
        );
    };

    const renderFields = (parameter: Parameter) => {
        if (parameter.enum) {
            return renderSelectField(parameter);
        }
        return renderInputField(parameter);
    };

    const renderInputContentIfApplicable = () => {
        if (parameters.length === 0) {
            return <React.Fragment />;
        }
        return (
            <React.Fragment>
                <h2>Modify parameters</h2>
                {parameters.map(renderFields)}
                <br />
            </React.Fragment>
        );
    };

    return (
        <div>
            {renderInputContentIfApplicable()}
            <br />
            Are you sure you want to resume node {props.nodeId} ?
        </div>
    );
}

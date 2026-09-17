import React, {useRef} from 'react';

export interface ArbitraryParameter {
    id: number;
    name: string;
    value: string;
}

interface Props {
    parameters: ArbitraryParameter[];
    onChange: (parameters: ArbitraryParameter[]) => void;
}

export function ArbitraryParametersInput({parameters, onChange}: Props) {
    const nextID = useRef(0);

    const addParameter = () => {
        onChange([...parameters, {id: nextID.current++, name: '', value: ''}]);
    };

    const updateParameter = (id: number, changes: Partial<Pick<ArbitraryParameter, 'name' | 'value'>>) => {
        onChange(parameters.map(parameter => (parameter.id === id ? {...parameter, ...changes} : parameter)));
    };

    const removeParameter = (id: number) => {
        onChange(parameters.filter(parameter => parameter.id !== id));
    };

    return (
        <>
            {parameters.map((parameter, index) => {
                const rowNumber = index + 1;
                const nameInputID = `arbitrary-parameter-name-${parameter.id}`;
                const valueInputID = `arbitrary-parameter-value-${parameter.id}`;
                const nameErrorID = `arbitrary-parameter-name-error-${parameter.id}`;

                return (
                    <div className='row' key={parameter.id} role='group' aria-label={`Arbitrary parameter ${rowNumber}`} style={{marginTop: 14}}>
                        <div className={`columns small-12 medium-4 ${parameter.name ? '' : 'error'}`}>
                            <label htmlFor={nameInputID}>Name</label>
                            <input
                                id={nameInputID}
                                className='argo-field'
                                aria-label={`Arbitrary parameter ${rowNumber} name`}
                                aria-invalid={!parameter.name}
                                aria-describedby={!parameter.name ? nameErrorID : undefined}
                                required
                                value={parameter.name}
                                onChange={event => updateParameter(parameter.id, {name: event.target.value})}
                            />
                            {!parameter.name && (
                                <div id={nameErrorID} className='argo-form-row__error-msg'>
                                    Parameter name is required
                                </div>
                            )}
                        </div>
                        <div className='columns small-12 medium-7'>
                            <label htmlFor={valueInputID}>Value</label>
                            <textarea
                                id={valueInputID}
                                className='argo-field'
                                aria-label={`Arbitrary parameter ${rowNumber} value`}
                                value={parameter.value}
                                onChange={event => updateParameter(parameter.id, {value: event.target.value})}
                            />
                        </div>
                        <div className='columns small-12 medium-1' style={{paddingTop: 24}}>
                            <button
                                type='button'
                                className='argo-button argo-button--base-o'
                                aria-label={`Remove arbitrary parameter ${rowNumber}`}
                                title={`Remove arbitrary parameter ${rowNumber}`}
                                onClick={() => removeParameter(parameter.id)}>
                                <i className='fa fa-trash' aria-hidden='true' />
                            </button>
                        </div>
                    </div>
                );
            })}
            <button type='button' className='argo-button argo-button--base-o' style={{marginTop: 14}} onClick={addParameter}>
                <i className='fa fa-plus' aria-hidden='true' /> Add a parameter
            </button>
        </>
    );
}

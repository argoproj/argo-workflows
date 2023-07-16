import {Select, Tooltip} from 'argo-ui';
import * as React from 'react';
import {Parameter} from '../../../../models';
import {Utils} from '../../utils';

interface ParametersInputProps {
    parameters: Parameter[];
    onChange?: (parameters: Parameter[]) => void;
}

export class ParametersInput extends React.Component<ParametersInputProps, {parameters: Parameter[]}> {
    constructor(props: ParametersInputProps) {
        super(props);
        this.state = {parameters: props.parameters || []};
    }

    public render() {
        return (
            <>
                {this.props.parameters.map((parameter, index) => (
                    <div key={parameter.name + '_' + index} style={{marginBottom: 14}}>
                        <label>{parameter.name}</label>
                        {parameter.description && (
                            <Tooltip content={parameter.description}>
                                <i className='fa fa-question-circle' style={{marginLeft: 4}} />
                            </Tooltip>
                        )}
                        {(parameter.enum && this.displaySelectFieldForEnumValues(parameter)) || this.displayInputFieldForSingleValue(parameter)}
                    </div>
                ))}
            </>
        );
    }

    private displaySelectFieldForEnumValues(parameter: Parameter) {
        return (
            <Select
                key={parameter.name}
                value={Utils.getValueFromParameter(parameter)}
                options={parameter.enum.map(value => ({
                    value,
                    title: value
                }))}
                onChange={e => {
                    const newParameters: Parameter[] = this.state.parameters.map(p => ({
                        name: p.name,
                        value: p.name === parameter.name ? e.value : Utils.getValueFromParameter(p),
                        enum: p.enum
                    }));
                    this.setState({parameters: newParameters});
                    this.onParametersChange(newParameters);
                }}
            />
        );
    }

    private displayInputFieldForSingleValue(parameter: Parameter) {
        return (
            <textarea
                className='argo-field'
                value={Utils.getValueFromParameter(parameter)}
                onChange={e => {
                    const newParameters: Parameter[] = this.state.parameters.map(p => ({
                        name: p.name,
                        value: p.name === parameter.name ? e.target.value : Utils.getValueFromParameter(p),
                        enum: p.enum
                    }));
                    this.setState({parameters: newParameters});
                    this.onParametersChange(newParameters);
                }}
            />
        );
    }

    private onParametersChange(parameters: Parameter[]) {
        if (this.props.onChange) {
            this.props.onChange(parameters);
        }
    }
}

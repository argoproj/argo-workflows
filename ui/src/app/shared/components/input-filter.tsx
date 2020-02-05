import * as React from 'react';
import {Autocomplete} from '../../../../node_modules/argo-ui';
import {ErrorPanel} from './error-panel';

interface InputProps {
    value: string;
    placeholder: string;
    type: string;
    onChange: (input: string) => void;
}

interface InputState {
    input: string;
    localInputs: string[];
    error?: Error;
}

export class InputFilter extends React.Component<InputProps, InputState> {
    constructor(props: Readonly<InputProps>) {
        super(props);
        this.state = {
            input: props.value,
            localInputs: (localStorage.getItem(this.props.type + '_inputs') || '').split(',').filter(input => input !== '')
        };
    }

    private set input(input: string) {
        this.setState(state => {
            const localInputs = state.localInputs;
            if (!state.localInputs.includes(input)) {
                localInputs.unshift(input);
            }
            while (localInputs.length > 5) {
                localInputs.pop();
            }
            localStorage.setItem(this.props.type + '_inputs', localInputs.join(','));
            return {input, localInputs};
        });
    }

    public render() {
        if (this.state.error) {
            return <ErrorPanel error={this.state.error} />;
        }
        return (
            <>
                <Autocomplete
                    items={this.state.localInputs}
                    value={this.state.input}
                    onChange={(e, input) => this.setState({input})}
                    onSelect={input => {
                        this.setState({input});
                        this.props.onChange(input);
                    }}
                    renderInput={inputProps => (
                        <input
                            {...inputProps}
                            onKeyUp={event => {
                                if (event.keyCode === 13) {
                                    this.input = event.currentTarget.value;
                                    this.props.onChange(this.state.input);
                                }
                            }}
                            className='argo-field'
                            placeholder={this.props.placeholder}
                        />
                    )}
                />
                <a
                    onClick={() => {
                        this.setState({input: ''});
                        this.props.onChange('');
                    }}>
                    <i className='fa fa-times-circle' />
                </a>
            </>
        );
    }
}

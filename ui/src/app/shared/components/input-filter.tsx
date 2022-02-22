import {Autocomplete} from 'argo-ui';
import * as React from 'react';

interface InputProps {
    value: string;
    placeholder?: string;
    name: string;
    onChange: (input: string) => void;
}

interface InputState {
    value: string;
    localCache: string[];
    error?: Error;
}

export class InputFilter extends React.Component<InputProps, InputState> {
    constructor(props: Readonly<InputProps>) {
        super(props);
        this.state = {
            value: props.value,
            localCache: (localStorage.getItem(this.props.name + '_inputs') || '').split(',').filter(value => value !== '')
        };
    }

    public render() {
        return (
            <>
                <Autocomplete
                    items={this.state.localCache}
                    value={this.state.value}
                    onChange={(e, value) => this.setState({value})}
                    onSelect={value => {
                        this.setState({value});
                        this.props.onChange(value);
                    }}
                    renderInput={inputProps => (
                        <input
                            {...inputProps}
                            onKeyUp={event => {
                                if (event.keyCode === 13) {
                                    this.setValueAndCache(event.currentTarget.value);
                                    this.props.onChange(this.state.value);
                                }
                            }}
                            className='argo-field'
                            placeholder={this.props.placeholder}
                        />
                    )}
                />
                <a
                    onClick={() => {
                        this.setState({value: ''});
                        this.props.onChange('');
                    }}>
                    <i className='fa fa-times-circle' />
                </a>
            </>
        );
    }

    private setValueAndCache(value: string) {
        this.setState(state => {
            const localCache = state.localCache;
            if (!state.localCache.includes(value)) {
                localCache.unshift(value);
            }
            while (localCache.length > 5) {
                localCache.pop();
            }
            localStorage.setItem(this.props.name + '_inputs', localCache.join(','));
            return {value, localCache};
        });
    }
}

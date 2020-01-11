import * as React from 'react';
import {Autocomplete} from '../../../../node_modules/argo-ui';

interface Props {
    value: string;
    onChange: (namespace: string) => void;
}

interface State {
    namespace: string;
    namespaces: string[];
}

export class NamespaceFilter extends React.Component<Props, State> {
    constructor(props: Readonly<Props>) {
        super(props);
        this.state = {
            namespace: props.value,
            namespaces: (localStorage.getItem('namespaces') || '').split(',').filter(ns => ns !== '')
        };
    }

    private set namespace(namespace: string) {
        this.setState(state => {
            const namespaces = state.namespaces;
            if (!state.namespaces.includes(namespace)) {
                namespaces.unshift(namespace);
            }
            while (namespaces.length > 5) {
                namespaces.pop();
            }
            localStorage.setItem('namespaces', namespaces.join(','));
            return {namespace, namespaces};
        });
    }

    public render() {
        return (
            <>
                <small>Namespace</small>{' '}
                <Autocomplete
                    items={this.state.namespaces}
                    value={this.state.namespace}
                    onChange={(e, namespace) => this.setState({namespace})}
                    onSelect={namespace => {
                        this.setState({namespace});
                        this.props.onChange(namespace);
                    }}
                    renderInput={inputProps => (
                        <input
                            {...inputProps}
                            onKeyUp={event => {
                                if (event.keyCode === 13) {
                                    this.namespace = event.currentTarget.value;
                                    this.props.onChange(this.state.namespace);
                                }
                            }}
                            className='argo-field'
                        />
                    )}
                />
                <a
                    onClick={() => {
                        this.setState({namespace: ''});
                        this.props.onChange('');
                    }}>
                    <i className='fa fa-times-circle' />
                </a>
            </>
        );
    }
}

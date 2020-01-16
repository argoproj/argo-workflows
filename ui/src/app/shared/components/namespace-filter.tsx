import * as React from 'react';
import {Autocomplete} from '../../../../node_modules/argo-ui';
import {services} from '../services';
import {ErrorPanel} from './error-panel';

interface Props {
    value: string;
    onChange: (namespace: string) => void;
}

interface State {
    editable: boolean;
    namespace: string;
    namespaces: string[];
    error?: Error;
}

export class NamespaceFilter extends React.Component<Props, State> {
    constructor(props: Readonly<Props>) {
        super(props);
        this.state = {
            editable: false,
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

    public componentDidMount(): void {
        services.info
            .get()
            .then(info => {
                if (info.managedNamespace && info.managedNamespace !== this.namespace) {
                    this.setState({editable: false, namespace: info.managedNamespace});
                    this.props.onChange(info.managedNamespace);
                } else {
                    this.setState({editable: true});
                }
            })
            .catch(error => this.setState({error}));
    }

    public render() {
        if (this.state.error) {
            return <ErrorPanel error={this.state.error} />;
        }
        if (!this.state.editable) {
            return <>{this.state.namespace}</>;
        }
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

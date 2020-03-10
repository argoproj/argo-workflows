import * as React from 'react';
import {services} from '../services';
import {ErrorPanel} from './error-panel';
import {InputFilter} from './input-filter';

interface Props {
    value: string;
    onChange: (namespace: string) => void;
}

interface State {
    editable: boolean;
    namespace: string;
    error?: Error;
}

export class NamespaceFilter extends React.Component<Props, State> {
    constructor(props: Readonly<Props>) {
        super(props);
        this.state = {
            editable: false,
            namespace: props.value
        };
    }

    public componentDidMount(): void {
        services.info
            .get()
            .then(info => {
                if (info.managedNamespace) {
                    const namespaceChanged = info.managedNamespace !== this.state.namespace;
                    this.setState({editable: false, namespace: info.managedNamespace});
                    if (namespaceChanged) {
                        this.props.onChange(info.managedNamespace);
                    }
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
            <InputFilter
                value={this.state.namespace}
                placeholder='Namespace'
                name='ns'
                onChange={ns => {
                    this.setState({namespace: ns});
                    this.props.onChange(ns);
                }}
            />
        );
    }
}

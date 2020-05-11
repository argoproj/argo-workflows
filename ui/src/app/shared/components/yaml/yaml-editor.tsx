import * as jsYaml from 'js-yaml';
import * as React from 'react';

import {YamlViewer} from './yaml-viewer';
require('./yaml.scss');
interface Props<T> {
    title?: string;
    value: T;
    editing: boolean;
    onSubmit: (value: T) => void;
}

interface State {
    editing: boolean;
    value: string;
    error?: Error;
}

export class YamlEditor<T> extends React.Component<Props<T>, State> {
    constructor(props: Readonly<Props<T>>) {
        super(props);
        this.state = {editing: this.props.editing, value: jsYaml.dump(this.props.value)};
    }

    public componentDidUpdate(prevProps: Props<T>) {
        if (prevProps.value !== this.props.value) {
            this.setState({
                value: jsYaml.dump(this.props.value)
            });
        }
    }

    public render() {
        return (
            <>
                {this.props.title && <h4>{this.props.title}</h4>}
                {this.renderButtons()}
                {this.state.error && (
                    <p>
                        <i className='fa fa-exclamation-triangle status-icon--failed' /> {this.state.error.message}
                    </p>
                )}
                {this.state.editing ? (
                    <textarea
                        className='yaml'
                        value={this.state.value}
                        onChange={e => this.setState({value: e.currentTarget.value})}
                        onFocus={e => (e.currentTarget.style.height = e.currentTarget.scrollHeight + 'px')}
                        autoFocus={true}
                    />
                ) : (
                    <YamlViewer value={this.state.value} />
                )}
            </>
        );
    }

    private renderButtons() {
        return (
            <div>
                {(this.state.editing && (
                    <button onClick={() => this.submit()} className='argo-button argo-button--base'>
                        Submit
                    </button>
                )) || (
                    <button onClick={() => this.setState({editing: true})} className='argo-button argo-button--base'>
                        Edit
                    </button>
                )}
            </div>
        );
    }

    private submit() {
        try {
            this.props.onSubmit(jsYaml.load(this.state.value));
            this.setState({editing: false});
        } catch (error) {
            this.setState({error});
        }
    }
}

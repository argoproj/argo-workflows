import * as jsYaml from 'js-yaml';
import * as monacoEditor from 'monaco-editor';
import * as React from 'react';

import {MonacoEditor} from '../monaco-editor';
import {YamlViewer} from './yaml-viewer/yaml-viewer';

interface Props<T> {
    title?: string;
    value: T;
    editing: boolean;
    onSubmit: (value: T) => void;
}

interface State {
    editing: boolean;
    error?: Error;
}

export class YamlEditor<T> extends React.Component<Props<T>, State> {
    private model: monacoEditor.editor.ITextModel;

    constructor(props: Readonly<Props<T>>) {
        super(props);
        this.state = {editing: this.props.editing};
    }

    public render() {
        const text = jsYaml.dump(this.props.value);
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
                    <MonacoEditor
                        editor={{
                            input: {text, language: 'yaml'},
                            options: {readOnly: !this.state.editing, minimap: {enabled: false}},
                            getApi: api => {
                                this.model = api.getModel() as monacoEditor.editor.ITextModel;
                            }
                        }}
                    />
                ) : (
                    <YamlViewer yaml={text} />
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
            this.props.onSubmit(jsYaml.load(this.model.getLinesContent().join('\n')));
            this.setState({editing: false});
        } catch (error) {
            this.setState({error});
        }
    }
}

import * as React from 'react';
import MonacoEditor from 'react-monaco-editor';
import {uiUrl} from '../../base';
import {ResourceViewer} from './resource-viewer';

import {languages} from 'monaco-editor/esm/vs/editor/editor.api';
import {parse, stringify} from './resource';

require('./resource.scss');

interface Props<T> {
    kind: string;
    upload?: boolean;
    title?: string;
    value: T;
    editing?: boolean;
    onSubmit?: (value: T) => void;
}

interface State {
    editing: boolean;
    value: string;
    error?: Error;
}

export class ResourceEditor<T> extends React.Component<Props<T>, State> {
    constructor(props: Readonly<Props<T>>) {
        super(props);
        this.state = {editing: this.props.editing, value: stringify(this.props.value)};
    }

    public componentDidMount() {
        const uri = uiUrl('assets/schemas/' + this.props.kind + '.json');
        fetch(uri)
            .then(res => res.json())
            .then(schema => {
                languages.json.jsonDefaults.setDiagnosticsOptions({
                    validate: true,
                    schemas: [
                        {
                            uri,
                            fileMatch: ['*'], // associate with our model
                            schema
                        }
                    ]
                });
            })
            .catch(error => this.setState({error}));
    }

    public componentDidUpdate(prevProps: Props<T>) {
        if (prevProps.value !== this.props.value) {
            this.setState({value: stringify(this.props.value)});
        }
    }

    public handleFiles(files: FileList) {
        files[0]
            .text()
            .then(value => {
                this.setState({value: stringify(parse(value))});
            })
            .catch(error => this.setState(error));
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
                    <MonacoEditor
                        value={this.state.value}
                        language='json'
                        height={'600px'}
                        onChange={value => this.setState({value})}
                        options={{extraEditorClassName: 'resource', minimap: {enabled: false}, lineNumbers: 'off', renderIndentGuides: false}}
                    />
                ) : (
                    <ResourceViewer value={parse(this.state.value)} />
                )}
            </>
        );
    }

    private renderButtons() {
        return (
            <div>
                {(this.state.editing && (
                    <>
                        {this.props.upload && (
                            <input
                                type='file'
                                onChange={e => {
                                    this.handleFiles(e.target.files);
                                }}
                            />
                        )}
                        <button onClick={() => this.submit()} className='argo-button argo-button--base'>
                            <i className='fa fa-plus' /> Submit
                        </button>
                    </>
                )) || (
                    <button onClick={() => this.setState({editing: true})} className='argo-button argo-button--base'>
                        <i className='fa fa-edit' /> Edit
                    </button>
                )}
            </div>
        );
    }

    private submit() {
        try {
            this.props.onSubmit(parse(this.state.value));
            this.setState({editing: false});
        } catch (error) {
            this.setState({error});
        }
    }
}

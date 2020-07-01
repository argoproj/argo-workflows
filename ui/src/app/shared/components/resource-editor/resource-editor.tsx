import * as React from 'react';
import MonacoEditor from 'react-monaco-editor';
import {uiUrl} from '../../base';

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
    type: string;
    value: string;
    error?: Error;
}

export class ResourceEditor<T> extends React.Component<Props<T>, State> {
    constructor(props: Readonly<Props<T>>) {
        super(props);
        this.state = {editing: this.props.editing, type: 'json', value: stringify(this.props.value, 'json')};
    }

    private set type(type: string) {
        this.setState({type, value: stringify(parse(this.state.value), type)});
    }

    public componentDidMount() {
        const uri = uiUrl('assets/openapi-spec/swagger.json');
        fetch(uri)
            .then(res => res.json())
            .then(swagger => {
                // adds auto-completion to JSON only
                languages.json.jsonDefaults.setDiagnosticsOptions({
                    validate: true,
                    schemas: [
                        {
                            uri,
                            fileMatch: ['*'],
                            schema: {
                                $id: 'http://workflows.argoproj.io/' + this.props.kind + '.json',
                                $ref: '#/definitions/io.argoproj.workflow.v1alpha1.' + this.props.kind,
                                $schema: 'http://json-schema.org/draft-07/schema',
                                definitions: swagger.definitions
                            }
                        }
                    ]
                });
            })
            .catch(error => this.setState({error}));
    }

    public componentDidUpdate(prevProps: Props<T>) {
        if (prevProps.value !== this.props.value) {
            this.setState({value: stringify(this.props.value, this.state.type)});
        }
    }

    public handleFiles(files: FileList) {
        files[0]
            .text()
            .then(value => this.setState({value: stringify(parse(value), this.state.type)}))
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
                <MonacoEditor
                    value={this.state.value}
                    language={this.state.type}
                    height={'600px'}
                    onChange={value => this.setState({value})}
                    options={{
                        readOnly: !this.state.editing,
                        extraEditorClassName: 'resource',
                        minimap: {enabled: false},
                        lineNumbers: 'off',
                        renderIndentGuides: false
                    }}
                />
                {this.renderFooter()}
            </>
        );
    }

    private renderButtons() {
        return (
            <div>
                {(this.state.editing && (
                    <>
                        <label className='argo-button argo-button--base-o'>
                            <input type={'checkbox'} checked={this.state.type === 'yaml'} onChange={e => (this.type = e.target.checked ? 'yaml' : 'json')} /> YAML
                        </label>{' '}
                        {this.props.upload && (
                            <label className='argo-button argo-button--base-o'>
                                <input type='file' onChange={e => this.handleFiles(e.target.files)} style={{display: 'none'}} />
                                <i className='fa fa-upload' /> Upload file
                            </label>
                        )}{' '}
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

    private renderFooter() {
        return this.state.editing ? (
            <small>
                <i className='fa fa-info-circle' /> {this.state.type === 'json' ? <>Full auto-completion</> : <>Basic completion</>} for {this.state.type.toUpperCase()}
            </small>
        ) : null;
    }
}

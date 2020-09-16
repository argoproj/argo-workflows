import * as React from 'react';
import MonacoEditor from 'react-monaco-editor';
import {uiUrl} from '../../base';

import {languages} from 'monaco-editor/esm/vs/editor/editor.api';
import {ErrorNotice} from '../error-notice';
import {ToggleButton} from '../toggle-button';
import {parse, stringify} from './resource';

require('./resource.scss');

interface Props<T> {
    kind: string;
    upload?: boolean;
    title?: string;
    value: T;
    readonly?: boolean;
    editing?: boolean;
    onSubmit?: (value: T) => Promise<any>;
}

interface State {
    editing: boolean;
    lang: string;
    value: string;
    error?: Error;
}

const LOCAL_STORAGE_KEY = 'ResourceEditorLang';

export class ResourceEditor<T> extends React.Component<Props<T>, State> {
    private set lang(lang: string) {
        try {
            this.setState(state => ({lang, error: null, value: stringify(parse(state.value), lang)}));
        } catch (error) {
            this.setState({error});
        }
    }

    private static saveLang(newLang: string) {
        localStorage.setItem(LOCAL_STORAGE_KEY, newLang);
    }

    private static loadLang(): string {
        const stored = localStorage.getItem(LOCAL_STORAGE_KEY);
        if (stored !== null) {
            if (stored === 'yaml' || stored === 'json') {
                return stored;
            }
        }
        return 'yaml';
    }

    constructor(props: Readonly<Props<T>>) {
        super(props);
        const storedLang = ResourceEditor.loadLang();
        this.state = {editing: this.props.editing, lang: storedLang, value: stringify(this.props.value, storedLang)};
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
            this.setState(state => ({value: stringify(this.props.value, state.lang)}));
        }
    }

    public handleFiles(files: FileList) {
        files[0]
            .text()
            .then(value => stringify(parse(value), this.state.lang))
            .then(value => this.setState({error: null, value}))
            .catch(error => this.setState(error));
    }

    public render() {
        return (
            <>
                {this.props.title && <h4>{this.props.title}</h4>}
                {!this.props.readonly && this.renderButtons()}
                {this.state.error && <ErrorNotice error={this.state.error} />}
                <div className='resource-editor-panel__editor'>
                    <MonacoEditor
                        key='editor'
                        value={this.state.value}
                        language={this.state.lang}
                        height={'600px'}
                        onChange={value => this.setState({value})}
                        options={{
                            readOnly: this.props.readonly || !this.state.editing,
                            extraEditorClassName: 'resource',
                            minimap: {enabled: false},
                            lineNumbers: 'off',
                            renderIndentGuides: false
                        }}
                    />
                </div>
                {this.renderWarning()}
            </>
        );
    }

    private changeLang() {
        const newLang = this.state.lang === 'yaml' ? 'json' : 'yaml';
        this.lang = newLang;
        ResourceEditor.saveLang(newLang);
    }

    private renderButtons() {
        return (
            <div>
                {(this.state.editing && (
                    <>
                        <ToggleButton toggled={this.state.lang === 'yaml'} onToggle={() => this.changeLang()}>
                            YAML
                        </ToggleButton>
                        {this.props.upload && (
                            <label className='argo-button argo-button--base-o' key='upload-file'>
                                <input type='file' onChange={e => this.handleFiles(e.target.files)} style={{display: 'none'}} />
                                <i className='fa fa-upload' /> Upload file
                            </label>
                        )}
                        <button onClick={() => this.submit()} className='argo-button argo-button--base' key='submit'>
                            <i className='fa fa-plus' /> Submit
                        </button>
                    </>
                )) || (
                    <button onClick={() => this.setState({editing: true})} className='argo-button argo-button--base' key='edit'>
                        <i className='fa fa-edit' /> Edit
                    </button>
                )}
            </div>
        );
    }

    private submit() {
        try {
            this.props
                .onSubmit(parse(this.state.value))
                .then(() => this.setState({error: null}))
                .catch(error => this.setState({error}));
        } catch (error) {
            this.setState({error});
        }
    }

    private renderWarning() {
        return (
            <div style={{marginTop: '1em'}}>
                <i className='fa fa-info-circle' />{' '}
                {this.state.lang === 'json' ? <>Full auto-completion enabled.</> : <>Basic completion for YAML. Switch to JSON for full auto-completion.</>}{' '}
                <a href='https://argoproj.github.io/argo/ide-setup/'>Learn how to get auto-completion in your IDE.</a>
            </div>
        );
    }
}

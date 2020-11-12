import * as React from 'react';
import {Button} from '../button';
import {ErrorNotice} from '../error-notice';
import {ToggleButton} from '../toggle-button';
import {ObjectEditor} from './object-editor';
import {parse, stringify} from './resource';

require('./resource.scss');

interface Props<T> {
    kind?: string;
    upload?: boolean;
    namespace?: string;
    title?: string;
    value: T;
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
                {this.renderButtons()}
                {this.state.error && <ErrorNotice error={this.state.error} />}
                <div className='resource-editor-panel__editor'>
                    <ObjectEditor
                        key='editor'
                        type={'io.argoproj.workflow.v1alpha1.' + this.props.kind}
                        value={this.state.value}
                        language={this.state.lang}
                        onChange={value => this.setState({value})}
                        onError={error => this.setState({error})}
                    />
                </div>
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
                <ToggleButton toggled={this.state.lang === 'yaml'} onToggle={() => this.changeLang()}>
                    YAML
                </ToggleButton>
                {this.state.editing ? (
                    <>
                        {this.props.upload && (
                            <label className='argo-button argo-button--base-o' key='upload-file'>
                                <input type='file' onChange={e => this.handleFiles(e.target.files)} style={{display: 'none'}} />
                                <i className='fa fa-upload' /> Upload file
                            </label>
                        )}
                        {this.props.onSubmit && (
                            <Button icon='plus' onClick={() => this.submit()} key='submit'>
                                Submit
                            </Button>
                        )}
                    </>
                ) : (
                    <Button icon='edit' onClick={() => this.setState({editing: true})} key='edit'>
                        Edit
                    </Button>
                )}
            </div>
        );
    }

    private submit() {
        try {
            const value = parse(this.state.value);
            if (value.metadata && !value.metadata.namespace && this.props.namespace) {
                value.metadata.namespace = this.props.namespace;
            }
            this.props
                .onSubmit(value)
                .then(() => this.setState({error: null}))
                .catch(error => this.setState({error}));
        } catch (error) {
            this.setState({error});
        }
    }
}

import * as kubernetes from 'argo-ui/src/models/kubernetes';
import * as React from 'react';
import {Button} from '../button';
import {ErrorNotice} from '../error-notice';
import {ObjectEditor} from '../object-editor/object-editor';
import {ToggleButton} from '../toggle-button';
import {UploadButton} from '../upload-button';

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

interface State<T> {
    editing: boolean;
    lang: string;
    value: T;
    error?: Error;
}

const LOCAL_STORAGE_KEY = 'ResourceEditorLang';

export class ResourceEditor<T extends {metadata: kubernetes.ObjectMeta}> extends React.Component<Props<T>, State<T>> {
    private set lang(lang: string) {
        this.setState({lang, error: null}, () => localStorage.setItem(LOCAL_STORAGE_KEY, lang));
    }

    private get lang() {
        return this.state.lang;
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
        this.state = {editing: this.props.editing, lang: ResourceEditor.loadLang(), value: this.props.value};
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
                        language={this.lang}
                        onChange={value => this.setState({value})}
                        onError={error => this.setState({error})}
                    />
                </div>
            </>
        );
    }

    private changeLang() {
        this.lang = this.lang === 'yaml' ? 'json' : 'yaml';
    }

    private renderButtons() {
        return (
            <div>
                <ToggleButton toggled={this.lang === 'yaml'} onToggle={() => this.changeLang()}>
                    YAML
                </ToggleButton>
                {this.state.editing ? (
                    <>
                        {this.props.upload && <UploadButton<T> lang={this.state.lang} onUpload={value => this.setState({value})} onError={error => this.setState({error})} />}
                        {this.props.onSubmit && (
                            <Button icon='plus' onClick={() => this.submit()} key='submit'>
                                Submit
                            </Button>
                        )}
                    </>
                ) : (
                    this.props.onSubmit && (
                        <Button icon='edit' onClick={() => this.setState({editing: true})} key='edit'>
                            Edit
                        </Button>
                    )
                )}
            </div>
        );
    }

    private submit() {
        try {
            const value = this.state.value;
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

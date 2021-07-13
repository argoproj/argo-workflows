import * as kubernetes from 'argo-ui/src/models/kubernetes';
import * as React from 'react';
import {Button} from '../button';
import {ErrorNotice} from '../error-notice';
import {ObjectEditor} from '../object-editor/object-editor';
import {UploadButton} from '../upload-button';

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
    value: T;
    error?: Error;
}

export class ResourceEditor<T extends {metadata?: kubernetes.ObjectMeta}> extends React.Component<Props<T>, State<T>> {
    constructor(props: Readonly<Props<T>>) {
        super(props);
        this.state = {editing: this.props.editing, value: this.props.value};
    }

    public render() {
        return (
            <>
                {this.props.title && <h4>{this.props.title}</h4>}
                {this.state.error && <ErrorNotice error={this.state.error} />}
                <ObjectEditor
                    key='editor'
                    type={'io.argoproj.workflow.v1alpha1.' + this.props.kind}
                    value={this.state.value}
                    buttons={this.renderButtons()}
                    onChange={value => this.setState({value})}
                />
            </>
        );
    }

    private renderButtons() {
        return (
            <>
                {this.state.editing ? (
                    <>
                        {this.props.upload && <UploadButton<T> onUpload={value => this.setState({value})} onError={error => this.setState({error})} />}
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
            </>
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

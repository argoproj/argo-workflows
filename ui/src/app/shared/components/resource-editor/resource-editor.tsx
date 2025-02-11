import * as kubernetes from 'argo-ui/src/models/kubernetes';
import * as React from 'react';
import {useState} from 'react';
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

export function ResourceEditor<T extends {metadata?: kubernetes.ObjectMeta}>(props: Props<T>) {
    const [editing, setEditing] = useState(props.editing);
    const [value, setValue] = useState(props.value);
    const [error, setError] = useState<Error>();

    async function submit() {
        try {
            if (value.metadata && !value.metadata.namespace && props.namespace) {
                value.metadata.namespace = props.namespace;
            }
            await props.onSubmit(value);
            setError(null);
        } catch (newError) {
            setError(newError);
        }
    }

    return (
        <>
            {props.title && <h4>{props.title}</h4>}
            <ErrorNotice error={error} />
            <ObjectEditor
                key='editor'
                type={'io.argoproj.workflow.v1alpha1.' + props.kind}
                value={value}
                buttons={
                    <>
                        {editing ? (
                            <>
                                {props.upload && <UploadButton<T> onUpload={setValue} onError={setError} />}
                                {props.onSubmit && (
                                    <Button icon='plus' onClick={() => submit()} key='submit'>
                                        Submit
                                    </Button>
                                )}
                            </>
                        ) : (
                            props.onSubmit && (
                                <Button icon='edit' onClick={() => setEditing(true)} key='edit'>
                                    Edit
                                </Button>
                            )
                        )}
                    </>
                }
                onChange={setValue}
            />
        </>
    );
}

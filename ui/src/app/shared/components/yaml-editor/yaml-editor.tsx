import {ErrorNotification, NotificationType} from 'argo-ui';
import * as jsYaml from 'js-yaml';
import * as monacoEditor from 'monaco-editor';
import * as React from 'react';

import {Consumer} from '../../context';
import {MonacoEditor} from '../monaco-editor';

// const jsonMergePatch = require('json-merge-patch');
require('./yaml-editor.scss');

const placeholderWorkflow: string = `apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: hello-world-
spec:
  entrypoint: whalesay
  templates:
  - name: whalesay
    container:
      image: docker/whalesay:latest
      command: [cowsay]
      args: ["hello world"]
`;

export class YamlEditor<T> extends React.Component<
    {
        input?: T;
        hideModeButtons?: boolean;
        initialEditMode?: boolean;
        onSave: (wf: string) => void;
        onCancel?: () => any;
        minHeight?: number;
    }
> {
    private model: monacoEditor.editor.ITextModel;

    constructor(props: any) {
        super(props);
        this.state = {editing: props.initialEditMode};
    }

    public render() {
        const props = this.props;
        const yaml = props.input ? jsYaml.safeDump(props.input) : placeholderWorkflow;

        return (
            <div className='yaml-editor'>
                {!props.hideModeButtons && (
                    <div className='yaml-editor__buttons'>
                            <Consumer>
                                {ctx => (
                                    <React.Fragment>
                                        <button
                                            onClick={async () => {
                                                try {
                                                    const rawWf = jsYaml.load(this.model.getLinesContent().join('\n'));
                                                    this.props.onSave(JSON.stringify(rawWf || {}));
                                                } catch (e) {
                                                    ctx.notifications.show({
                                                        content: <ErrorNotification title='Unable to submit workflow' e={e} />,
                                                        type: NotificationType.Error
                                                    });
                                                }
                                            }}
                                            className='argo-button argo-button--base'>
                                            Submit
                                        </button>{' '}
                                        {/*<button*/}
                                        {/*    onClick={() => {*/}
                                        {/*        this.model.setValue(jsYaml.safeDump(props.input));*/}
                                        {/*        this.setState({editing: !this.state.editing});*/}
                                        {/*        if (props.onCancel) {*/}
                                        {/*            props.onCancel();*/}
                                        {/*        }*/}
                                        {/*    }}*/}
                                        {/*    className='argo-button argo-button--base-o'>*/}
                                        {/*    Cancel*/}
                                        {/*</button>*/}
                                    </React.Fragment>
                                )}
                            </Consumer>
                    </div>
                )}
                <MonacoEditor
                    minHeight={props.minHeight}
                    editor={{
                        input: {text: yaml, language: 'yaml'},
                        options: {minimap: {enabled: false}},
                        getApi: api => {
                            this.model = api.getModel() as monacoEditor.editor.ITextModel;
                        }
                    }}
                />
            </div>
        );
    }
}

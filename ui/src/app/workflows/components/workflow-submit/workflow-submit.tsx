import * as jsYaml from 'js-yaml';
import * as React from 'react';
import {Workflow} from '../../../../models';
import {YamlEditor} from '../../../shared/components/yaml-editor/yaml-editor';
import {services} from '../../../shared/services';

interface Props {
    placeholder: Workflow;
    onSaved: (workflow: Workflow) => void;
    onError: (error: Error) => void;
}

export default class WorkflowSubmit extends React.Component<Props> {
    constructor(props: Readonly<Props>) {
        super(props);
    }

    public render() {
        return (
            <>
                <h4> Submit Workflow</h4>
                <YamlEditor
                    minHeight={800}
                    initialEditMode={true}
                    submitMode={true}
                    placeHolder={jsYaml.dump(this.props.placeholder)}
                    onSave={rawWf => {
                        const workflow = JSON.parse(rawWf) as Workflow;
                        return services.workflows
                            .create(workflow, workflow.metadata.namespace)
                            .then(this.props.onSaved)
                            .catch(this.props.onError);
                    }}
                />
            </>
        );
    }
}

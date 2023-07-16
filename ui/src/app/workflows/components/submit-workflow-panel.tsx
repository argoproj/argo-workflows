import {Select} from 'argo-ui';
import * as React from 'react';
import {Parameter, Template, Workflow} from '../../../models';
import {uiUrl} from '../../shared/base';
import {ErrorNotice} from '../../shared/components/error-notice';
import {ParametersInput} from '../../shared/components/parameters-input/parameters-input';
import {TagsInput} from '../../shared/components/tags-input/tags-input';
import {services} from '../../shared/services';
import {Utils} from '../../shared/utils';

interface Props {
    kind: string;
    namespace: string;
    name: string;
    entrypoint: string;
    templates: Template[];
    workflowParameters: Parameter[];
}

interface State {
    entrypoint: string;
    entrypoints: string[];
    parameters: Parameter[];
    workflowParameters: Parameter[];
    selectedTemplate: Template;
    templates: Template[];
    labels: string[];
    error?: Error;
    isSubmitting: boolean;
}

const workflowEntrypoint = '<default>';

export class SubmitWorkflowPanel extends React.Component<Props, State> {
    constructor(props: any) {
        super(props);
        const defaultTemplate: Template = {
            name: workflowEntrypoint,
            inputs: {
                parameters: []
            }
        };
        const state: State = {
            entrypoint: workflowEntrypoint,
            entrypoints: this.props.templates.map(t => t.name),
            selectedTemplate: defaultTemplate,
            parameters: [] as Parameter[],
            workflowParameters: JSON.parse(JSON.stringify(this.props.workflowParameters)),
            templates: [defaultTemplate].concat(this.props.templates),
            labels: ['submit-from-ui=true'],
            isSubmitting: false
        };
        this.state = state;
    }

    public render() {
        return (
            <>
                <h4>Submit Workflow</h4>
                <h5>
                    {this.props.namespace}/{this.props.name}
                </h5>
                {this.state.error && <ErrorNotice error={this.state.error} />}
                <div className='white-box'>
                    <div key='entrypoint' title='Entrypoint' style={{marginBottom: 25}}>
                        <label>Entrypoint</label>
                        <Select
                            value={this.state.entrypoint}
                            options={this.state.templates.map(t => ({
                                value: t.name,
                                title: t.name
                            }))}
                            onChange={selected => {
                                const selectedTemplate = this.getSelectedTemplate(selected.value);
                                this.setState({
                                    entrypoint: selected.value,
                                    selectedTemplate,
                                    parameters: (selectedTemplate && selectedTemplate.inputs.parameters) || []
                                });
                            }}
                        />
                    </div>
                    <div key='parameters' style={{marginBottom: 25}}>
                        <label>Parameters</label>
                        {this.state.workflowParameters.length > 0 && (
                            <ParametersInput parameters={this.state.workflowParameters} onChange={workflowParameters => this.setState({workflowParameters})} />
                        )}
                        {this.state.parameters.length > 0 && <ParametersInput parameters={this.state.parameters} onChange={parameters => this.setState({parameters})} />}
                        {this.state.workflowParameters.length === 0 && this.state.parameters.length === 0 ? (
                            <>
                                <br />
                                <label>No parameters</label>
                            </>
                        ) : (
                            <></>
                        )}
                    </div>
                    <div key='labels' style={{marginBottom: 25}}>
                        <label>Labels</label>
                        <TagsInput tags={this.state.labels} onChange={labels => this.setState({labels})} />
                    </div>
                    <div key='submit'>
                        <button onClick={() => this.submit()} className='argo-button argo-button--base' disabled={this.state.isSubmitting}>
                            <i className='fa fa-plus' /> {this.state.isSubmitting ? 'Loading...' : 'Submit'}
                        </button>
                    </div>
                </div>
            </>
        );
    }

    private getSelectedTemplate(entrypoint: string): Template {
        for (const t of this.state.templates) {
            if (t.name === entrypoint) {
                return t;
            }
        }
        return null;
    }

    private submit() {
        this.setState({isSubmitting: true});
        services.workflows
            .submit(this.props.kind, this.props.name, this.props.namespace, {
                entryPoint: this.state.entrypoint === workflowEntrypoint ? null : this.state.entrypoint,
                parameters: [
                    ...this.state.workflowParameters.filter(p => Utils.getValueFromParameter(p) !== undefined).map(p => p.name + '=' + Utils.getValueFromParameter(p)),
                    ...this.state.parameters.filter(p => Utils.getValueFromParameter(p) !== undefined).map(p => p.name + '=' + Utils.getValueFromParameter(p))
                ],
                labels: this.state.labels.join(',')
            })
            .then((submitted: Workflow) => (document.location.href = uiUrl(`workflows/${submitted.metadata.namespace}/${submitted.metadata.name}`)))
            .catch(error => this.setState({error, isSubmitting: false}));
    }
}

import * as React from 'react';
import {Parameter, Template, Workflow} from '../../../models';
import {uiUrl} from '../../shared/base';
import {ErrorNotice} from '../../shared/components/error-notice';
import {services} from '../../shared/services';

import {Select} from 'argo-ui';
import {TagsInput} from '../../shared/components/tags-input/tags-input';

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
    selectedTemplate: Template;
    templates: Template[];
    labels: string[];
    error?: Error;
}

const workflowEntrypoint = '<default>';

export class SubmitWorkflowPanel extends React.Component<Props, State> {
    constructor(props: any) {
        super(props);
        const defaultTemplate: Template = {
            name: workflowEntrypoint,
            inputs: {
                parameters: this.props.workflowParameters
            }
        };
        const state = {
            entrypoint: workflowEntrypoint,
            entrypoints: this.props.templates.map(t => t.name),
            selectedTemplate: defaultTemplate,
            parameters: this.props.workflowParameters || [],
            templates: [defaultTemplate].concat(this.props.templates),
            labels: ['submit-from-ui=true']
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
                        {this.state.parameters.length > 0 ? (
                            <>
                                {this.state.parameters.map((parameter, index) => (
                                    <div key={parameter.name + '_' + index}>
                                        <label>{parameter.name}</label>
                                        {(parameter.enum && this.displaySelectFieldForEnumValues(parameter)) || this.displayInputFieldForSingleValue(parameter)}
                                    </div>
                                ))}
                            </>
                        ) : (
                            <>
                                <br />
                                <label>No parameters</label>
                            </>
                        )}
                    </div>
                    <div key='labels' style={{marginBottom: 25}}>
                        <label>Labels</label>
                        <TagsInput tags={this.state.labels} onChange={labels => this.setState({labels})} />
                    </div>
                    <div key='submit'>
                        <button onClick={() => this.submit()} className='argo-button argo-button--base'>
                            <i className='fa fa-plus' /> Submit
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

    private displaySelectFieldForEnumValues(parameter: Parameter) {
        return (
            <Select
                key={parameter.name}
                value={parameter.value}
                options={parameter.enum.map(value => ({
                    value,
                    title: value
                }))}
                onChange={event => {
                    this.setState({
                        parameters: this.state.parameters.map(p => ({
                            name: p.name,
                            value: p.name === parameter.name ? event.value : p.value,
                            enum: p.enum
                        }))
                    });
                }}
            />
        );
    }

    private displayInputFieldForSingleValue(parameter: Parameter) {
        return (
            <input
                className='argo-field'
                value={parameter.value}
                onChange={event => {
                    this.setState({
                        parameters: this.state.parameters.map(p => ({
                            name: p.name,
                            value: p.name === parameter.name ? event.target.value : p.value,
                            enum: p.enum
                        }))
                    });
                }}
            />
        );
    }

    private submit() {
        services.workflows
            .submit(this.props.kind, this.props.name, this.props.namespace, {
                entryPoint: this.state.entrypoint === workflowEntrypoint ? null : this.state.entrypoint,
                parameters: this.state.parameters.map(p => p.name + '=' + p.value),
                labels: this.state.labels.join(',')
            })
            .then((submitted: Workflow) => (document.location.href = uiUrl(`workflows/${submitted.metadata.namespace}/${submitted.metadata.name}`)))
            .catch(error => this.setState({error}));
    }
}

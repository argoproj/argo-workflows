import * as React from 'react';
import {Parameter, Workflow} from '../../../models';
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
    entrypoints: string[];
    parameters: Parameter[];
}

interface State {
    entrypoint: string;
    parameters: Parameter[];
    labels: string[];
    error?: Error;
}

export class SubmitWorkflowPanel extends React.Component<Props, State> {
    constructor(props: any) {
        super(props);
        this.state = {
            entrypoint: this.props.entrypoint || (this.props.entrypoints.length > 0 && this.props.entrypoints[0]),
            parameters: this.props.parameters,
            labels: ['submit-from-ui=true']
        };
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
                            options={this.props.entrypoints.map((value, index) => ({
                                value,
                                title: value
                            }))}
                            onChange={selected => this.setState({entrypoint: selected.value})}
                        />
                    </div>
                    <div key='parameters' style={{marginBottom: 25}}>
                        <label>Parameters</label>
                        {this.state.parameters.length > 0 ? (
                            <>
                                {this.state.parameters.map(parameter => (
                                    <p key={parameter.name}>
                                        <label>
                                            {parameter.name}
                                            <input
                                                className='argo-field'
                                                value={parameter.value}
                                                onChange={event => {
                                                    this.setState({
                                                        parameters: this.state.parameters.map(p => ({
                                                            name: p.name,
                                                            value: p.name === parameter.name ? event.target.value : p.value
                                                        }))
                                                    });
                                                }}
                                            />
                                        </label>
                                    </p>
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

    private submit() {
        services.workflows
            .submit(this.props.kind, this.props.name, this.props.namespace, {
                entryPoint: this.state.entrypoint,
                parameters: this.state.parameters.map(p => p.name + '=' + p.value),
                labels: this.state.labels.join(',')
            })
            .then((submitted: Workflow) => (document.location.href = uiUrl(`workflows/${submitted.metadata.namespace}/${submitted.metadata.name}`)))
            .catch(error => this.setState({error}));
    }
}

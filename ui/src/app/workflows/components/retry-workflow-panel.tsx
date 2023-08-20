import {Checkbox} from 'argo-ui';
import * as React from 'react';
import {Parameter, RetryOpts, Workflow} from '../../../models';
import {uiUrl} from '../../shared/base';
import {ErrorNotice} from '../../shared/components/error-notice';
import {ParametersInput} from '../../shared/components/parameters-input/parameters-input';
import {services} from '../../shared/services';
import {Utils} from '../../shared/utils';

interface Props {
    workflow: Workflow;
    isArchived: boolean;
}

interface State {
    overrideParameters: boolean;
    restartSuccessful: boolean;
    workflowParameters: Parameter[];
    nodeFieldSelector: string;
    error?: Error;
    isSubmitting: boolean;
}

export class RetryWorkflowPanel extends React.Component<Props, State> {
    constructor(props: any) {
        super(props);
        const state: State = {
            workflowParameters: JSON.parse(JSON.stringify(this.props.workflow.spec.arguments.parameters || [])),
            isSubmitting: false,
            overrideParameters: false,
            nodeFieldSelector: '',
            restartSuccessful: false
        };
        this.state = state;
    }

    public render() {
        return (
            <>
                <h4>Retry Workflow</h4>
                <h5>
                    {this.props.workflow.metadata.namespace}/{this.props.workflow.metadata.name}
                </h5>

                {this.state.error && <ErrorNotice error={this.state.error} />}
                <div className='white-box'>
                    <div key='override-parameters' style={{marginBottom: 25}}>
                        <label>Override Parameters</label>
                        <div className='columns small-9'>
                            <Checkbox checked={this.state.overrideParameters} onChange={overrideParameters => this.setState({overrideParameters})} />
                        </div>
                    </div>

                    {this.state.overrideParameters && (
                        <div key='parameters' style={{marginBottom: 25}}>
                            <label>Parameters</label>
                            {this.state.workflowParameters.length > 0 && (
                                <ParametersInput parameters={this.state.workflowParameters} onChange={workflowParameters => this.setState({workflowParameters})} />
                            )}
                            {this.state.workflowParameters.length === 0 ? (
                                <>
                                    <br />
                                    <label>No parameters</label>
                                </>
                            ) : (
                                <></>
                            )}
                        </div>
                    )}

                    <div key='restart-successful' style={{marginBottom: 25}}>
                        <label>Restart Successful</label>
                        <div className='columns small-9'>
                            <Checkbox checked={this.state.restartSuccessful} onChange={restartSuccessful => this.setState({restartSuccessful})} />
                        </div>
                    </div>

                    {this.state.restartSuccessful && (
                        <div key='node-field-selector' style={{marginBottom: 25}}>
                            <label>
                                Node Field Selector to restart nodes. <a href='https://argoproj.github.io/argo-workflows/node-field-selector/'>See document</a>.
                            </label>

                            <div className='columns small-9'>
                                <textarea className='argo-field' value={this.state.nodeFieldSelector} onChange={e => this.setState({nodeFieldSelector: e.target.value})} />
                            </div>
                        </div>
                    )}

                    <div key='retry'>
                        <button onClick={() => this.submit()} className='argo-button argo-button--base' disabled={this.state.isSubmitting}>
                            <i className='fa fa-plus' /> {this.state.isSubmitting ? 'Loading...' : 'Retry'}
                        </button>
                    </div>
                </div>
            </>
        );
    }

    private submit() {
        this.setState({isSubmitting: true});
        const parameters: RetryOpts['parameters'] = this.state.overrideParameters
            ? [...this.state.workflowParameters.filter(p => Utils.getValueFromParameter(p) !== undefined).map(p => p.name + '=' + Utils.getValueFromParameter(p))]
            : [];
        const opts: RetryOpts = {
            parameters,
            restartSuccessful: this.state.restartSuccessful,
            nodeFieldSelector: this.state.nodeFieldSelector
        };

        if (!this.props.isArchived) {
            services.workflows
                .retry(this.props.workflow.metadata.name, this.props.workflow.metadata.namespace, opts)
                .then((submitted: Workflow) => (document.location.href = uiUrl(`workflows/${submitted.metadata.namespace}/${submitted.metadata.name}`)))
                .catch(error => this.setState({error, isSubmitting: false}));
        } else {
            services.workflows
                .retryArchived(this.props.workflow.metadata.uid, this.props.workflow.metadata.namespace, opts)
                .then((submitted: Workflow) => (document.location.href = uiUrl(`workflows/${submitted.metadata.namespace}/${submitted.metadata.name}`)))
                .catch(error => this.setState({error, isSubmitting: false}));
        }
    }
}

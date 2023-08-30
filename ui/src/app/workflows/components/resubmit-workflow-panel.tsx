import {Checkbox} from 'argo-ui';
import * as React from 'react';
import {Parameter, ResubmitOpts, Workflow} from '../../../models';
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
    workflowParameters: Parameter[];
    memoized: boolean;
    error?: Error;
    isSubmitting: boolean;
}

export class ResubmitWorkflowPanel extends React.Component<Props, State> {
    constructor(props: any) {
        super(props);
        const state: State = {
            workflowParameters: JSON.parse(JSON.stringify(this.props.workflow.spec.arguments.parameters || [])),
            memoized: false,
            isSubmitting: false,
            overrideParameters: false
        };
        this.state = state;
    }

    public render() {
        return (
            <>
                <h4>Resubmit Workflow</h4>
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

                    <div key='memoized' style={{marginBottom: 25}}>
                        <label>Memoized</label>
                        <div className='columns small-9'>
                            <Checkbox checked={this.state.memoized} onChange={memoized => this.setState({memoized})} />
                        </div>
                    </div>

                    {this.state.overrideParameters && this.state.memoized && (
                        <div key='warning-override-with-memoized'>
                            <i className='fa fa-exclamation-triangle' style={{color: '#f4c030'}} />
                            Overriding parameters on memoized submitted workflows may have unexpected results.
                        </div>
                    )}

                    <div key='resubmit'>
                        <button onClick={() => this.submit()} className='argo-button argo-button--base' disabled={this.state.isSubmitting}>
                            <i className='fa fa-plus' /> {this.state.isSubmitting ? 'Loading...' : 'Resubmit'}
                        </button>
                    </div>
                </div>
            </>
        );
    }

    private submit() {
        this.setState({isSubmitting: true});
        const parameters: ResubmitOpts['parameters'] = this.state.overrideParameters
            ? [...this.state.workflowParameters.filter(p => Utils.getValueFromParameter(p) !== undefined).map(p => p.name + '=' + Utils.getValueFromParameter(p))]
            : [];
        const opts: ResubmitOpts = {
            parameters,
            memoized: this.state.memoized
        };

        if (!this.props.isArchived) {
            services.workflows
                .resubmit(this.props.workflow.metadata.name, this.props.workflow.metadata.namespace, opts)
                .then((submitted: Workflow) => (document.location.href = uiUrl(`workflows/${submitted.metadata.namespace}/${submitted.metadata.name}`)))
                .catch(error => this.setState({error, isSubmitting: false}));
        } else {
            services.workflows
                .resubmitArchived(this.props.workflow.metadata.uid, this.props.workflow.metadata.namespace, opts)
                .then((submitted: Workflow) => (document.location.href = uiUrl(`workflows/${submitted.metadata.namespace}/${submitted.metadata.name}`)))
                .catch(error => this.setState({error, isSubmitting: false}));
        }
    }
}

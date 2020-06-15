import * as React from 'react';
import {Workflow} from '../../../../models';
import {AppContext, Consumer, ContextApis} from '../../../shared/context';
import {services} from '../../../shared/services';
import { NotificationType } from '../../../../../node_modules/argo-ui';

require('./workflows-toolbar.scss');

interface WorkflowsToolbarProps {
    selectedWorkflows: {[index: string]: Workflow};
    loadWorkflows: () => void;
    canSuspendSelected: boolean;
}

interface WorkflowsToolbarState {
    message: string;
}

export class WorkflowsToolbar extends React.Component<WorkflowsToolbarProps, WorkflowsToolbarState> {
    constructor(props: WorkflowsToolbarProps) {
        super(props);
        this.deleteSelectedWorkflows = this.deleteSelectedWorkflows.bind(this);
        this.state = {
            message: '',
        };
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <div className='workflows-toolbar'>
                        <div className='workflows-toolbar__count'>
                            {this.noneSelected() ? 'No' : this.getNumberSelected()}
                            &nbsp;workflow{this.getNumberSelected() === 1 ? '' : 's'} selected
                        </div>
                        <div className='workflows-toolbar__message'>{this.state.message}</div>
                        <div className='workflows-toolbar__actions'>
                            <button
                                onClick={() => this.deleteSelectedWorkflows(ctx)}
                                className='workflows-toolbar__actions--delete'
                                disabled={this.noneSelected()}>
                                <i className='fas fa-trash-alt' />
                                &nbsp;Delete Selected
                            </button>
                            <button
                                onClick={() => this.suspendSelectedWorkflows(ctx)}
                                className={'workflows-toolbar__actions--suspend'}
                                disabled={this.noneSelected() || !this.props.canSuspendSelected}>
                                <i className='fas fa-pause'></i>
                                &nbsp;Suspend Selected
                            </button>
                        </div>
                    </div>
                )}
            </Consumer>
        )
    }

    private noneSelected(): boolean {
        return Object.keys(this.props.selectedWorkflows).length < 1;
    }

    private getNumberSelected(): number {
        return Object.keys(this.props.selectedWorkflows).length;
    }

    private deleteSelectedWorkflows(ctx: ContextApis): void {
        if (!confirm('Are you sure you want to delete all selected workflows?')) {
            return;
        }
        for (const wfUID of Object.keys(this.props.selectedWorkflows)) {
            const wf = this.props.selectedWorkflows[wfUID];
            services.workflows
                .delete(wf.metadata.name, wf.metadata.namespace)
                .then(() => {
                    this.setState({message: 'Successfully deleted workflows'});
                    this.props.loadWorkflows();
                })
                .catch((err) => {
                    this.appContext.apis.notifications.show({
                        content: 'Unable to delete workflows',
                        type: NotificationType.Error
                    });
                });
        }
    }

    private suspendSelectedWorkflows(ctx: ContextApis): void {
        if (!confirm('Are you sure you want to suspend all selected workflows?')) {
            return;
        }
        for (const wfUID of Object.keys(this.props.selectedWorkflows)) {
            const wf = this.props.selectedWorkflows[wfUID];
            services.workflows
                .suspend(wf.metadata.name, wf.metadata.namespace)
                .then(() => {
                    this.setState({message: 'Successfully suspended workflows'});
                    this.props.loadWorkflows();
                })
                .catch((err) => {
                   this.setState({message: 'Unable to suspend workflows'})
                });
        }
    }

    private get appContext(): AppContext {
        return this.context as AppContext;
    }
}

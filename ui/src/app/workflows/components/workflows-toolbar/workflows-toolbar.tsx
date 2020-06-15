import * as React from 'react';
import {Workflow} from '../../../../models';
import {AppContext, Consumer, ContextApis} from '../../../shared/context';
import {services} from '../../../shared/services';
import { NotificationType } from '../../../../../node_modules/argo-ui';

require('./workflows-toolbar.scss');

interface WorkflowsToolbarProps {
    selectedWorkflows: {[index: string]: Workflow};
    loadWorkflows: () => void;
}

export class WorkflowsToolbar extends React.Component<WorkflowsToolbarProps, {message: string}> {
    constructor(props: WorkflowsToolbarProps) {
        super(props);
        this.deleteSelectedWorkflows = this.deleteSelectedWorkflows.bind(this);
        this.state = {
            message: ''
        };
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <div className='workflows-toolbar'>
                        <div className='workflows-toolbar__count'>{this.getNumberSelected()} workflows selected</div>
                        <div className='workflows-toolbar__message'>{this.state.message}</div>
                        <div className='workflows-toolbar__actions'>
                            <button onClick={() => this.deleteSelectedWorkflows(ctx)} className='workflows-toolbar__actions--delete'>
                                <i className='fas fa-trash-alt' />
                                &nbsp;Delete Selected
                            </button>
                            <button
                                onClick={() => this.suspendSelectedWorkflows(ctx)}
                                className={'workflows-toolbar__actions--suspend'}
                                disabled={false}>
                                <i className='fas fa-pause'/>
                                &nbsp;Suspend Selected
                            </button>
                        </div>
                    </div>
                )}
            </Consumer>
        )
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
                    this.setState({ message: 'Successfully deleted workflows'});
                    this.props.loadWorkflows()
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
                })
                .catch((err) => {
                   this.appContext.apis.notifications.show({
                        content: 'Unable to suspend workflows',
                        type: NotificationType.Error
                    });
                });
        }
    }

    private get appContext(): AppContext {
        return this.context as AppContext;
    }
}

import {AppContext, NotificationType, Page, SlidingPanel} from 'argo-ui';
import * as classNames from 'classnames';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {Subscription} from 'rxjs';

import * as models from '../../../../models';
import {NodePhase} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {services} from '../../../shared/services';

import {WorkflowArtifacts, WorkflowDag, WorkflowLogsViewer, WorkflowNodeInfo, WorkflowSummaryPanel, WorkflowTimeline, WorkflowYamlViewer} from '..';
import {Consumer, ContextApis} from '../../../shared/context';
import {Utils} from '../../../shared/utils';
import {WorkflowParametersPanel} from '../workflow-parameters-panel';

require('./workflow-details.scss');

function parseSidePanelParam(param: string) {
    const [type, nodeId, container] = (param || '').split(':');
    if (type === 'logs' || type === 'yaml') {
        return {type, nodeId, container: container || 'main'};
    }
    return null;
}

export class WorkflowDetails extends React.Component<RouteComponentProps<any>, {workflow: models.Workflow}> {
    public static contextTypes = {
        router: PropTypes.object,
        apis: PropTypes.object
    };

    private changesSubscription: Subscription;
    private timelineComponent: WorkflowTimeline;

    private get selectedTabKey() {
        return new URLSearchParams(this.props.location.search).get('tab') || 'workflow';
    }

    private get selectedNodeId() {
        return new URLSearchParams(this.props.location.search).get('nodeId');
    }

    private get sidePanel() {
        return parseSidePanelParam(new URLSearchParams(this.props.location.search).get('sidePanel'));
    }

    constructor(props: RouteComponentProps<any>) {
        super(props);
        this.state = {workflow: null};
    }

    public componentDidMount() {
        this.loadWorkflow(this.props.match.params.namespace, this.props.match.params.name);
    }

    public componentWillReceiveProps(nextProps: RouteComponentProps<any>) {
        if (this.props.match.params.name !== nextProps.match.params.name || this.props.match.params.namespace !== nextProps.match.params.namespace) {
            this.loadWorkflow(nextProps.match.params.namespace, nextProps.match.params.name);
        }
    }

    public componentDidUpdate(prevProps: RouteComponentProps<any>) {
        // Redraw timeline component after node details panel collapsed/expanded.
        const prevSelectedNodeId = new URLSearchParams(this.props.location.search).get('nodeId');
        if (this.timelineComponent && !!this.selectedNodeId !== !!prevSelectedNodeId) {
            setTimeout(() => {
                this.timelineComponent.updateWidth();
            }, 300);
        }
    }

    public componentWillUnmount() {
        this.ensureUnsubscribed();
    }

    public render() {
        const selectedNode = this.state.workflow && this.state.workflow.status && this.state.workflow.status.nodes && this.state.workflow.status.nodes[this.selectedNodeId];
        const workflowPhase: NodePhase = this.state.workflow && this.state.workflow.status ? this.state.workflow.status.phase : undefined;
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title={'Workflow Details'}
                        toolbar={{
                            breadcrumbs: [
                                {
                                    title: 'Workflows',
                                    path: uiUrl('workflows')
                                },
                                {title: this.props.match.params.name}
                            ],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Retry',
                                        iconClassName: 'fa fa-undo',
                                        disabled: workflowPhase === undefined || !(workflowPhase === 'Failed' || workflowPhase === 'Error'),
                                        action: () => this.retryWorkflow(ctx)
                                    },
                                    {
                                        title: 'Resubmit',
                                        iconClassName: 'fa fa-plus-circle ',
                                        action: () => this.resubmitWorkflow(ctx)
                                    },
                                    {
                                        title: 'Suspend',
                                        iconClassName: 'fa fa-pause',
                                        disabled: !Utils.isWorkflowRunning(this.state.workflow) || Utils.isWorkflowSuspended(this.state.workflow),
                                        action: () => this.suspendWorkflow(ctx)
                                    },
                                    {
                                        title: 'Resume',
                                        iconClassName: 'fa fa-play',
                                        disabled: !Utils.isWorkflowSuspended(this.state.workflow),
                                        action: () => this.resumeWorkflow(ctx)
                                    },
                                    {
                                        title: 'Terminate',
                                        iconClassName: 'fa fa-times-circle',
                                        disabled: !Utils.isWorkflowRunning(this.state.workflow),
                                        action: () => this.terminateWorkflow(ctx)
                                    },
                                    {
                                        title: 'Delete',
                                        iconClassName: 'fa fa-trash',
                                        action: () => this.deleteWorkflow(ctx)
                                    }
                                ]
                            },
                            tools: (
                                <div className='workflow-details__topbar-buttons'>
                                    <a className={classNames({active: this.selectedTabKey === 'summary'})} onClick={() => this.selectTab('summary')}>
                                        <i className='fa fa-columns' />
                                    </a>
                                    <a className={classNames({active: this.selectedTabKey === 'timeline'})} onClick={() => this.selectTab('timeline')}>
                                        <i className='fa argo-icon-timeline' />
                                    </a>
                                    <a className={classNames({active: this.selectedTabKey === 'workflow'})} onClick={() => this.selectTab('workflow')}>
                                        <i className='fa argo-icon-workflow' />
                                    </a>
                                </div>
                            )
                        }}>
                        <div className={classNames('workflow-details', {'workflow-details--step-node-expanded': !!selectedNode})}>
                            {(this.selectedTabKey === 'summary' && this.renderSummaryTab()) ||
                                (this.state.workflow && (
                                    <div>
                                        <div className='workflow-details__graph-container'>
                                            {(this.selectedTabKey === 'workflow' && (
                                                <WorkflowDag workflow={this.state.workflow} selectedNodeId={this.selectedNodeId} nodeClicked={node => this.selectNode(node.id)} />
                                            )) || (
                                                <WorkflowTimeline
                                                    workflow={this.state.workflow}
                                                    selectedNodeId={this.selectedNodeId}
                                                    nodeClicked={node => this.selectNode(node.id)}
                                                    ref={timeline => (this.timelineComponent = timeline)}
                                                />
                                            )}
                                        </div>
                                        <div className='workflow-details__step-info'>
                                            <button className='workflow-details__step-info-close' onClick={() => this.removeNodeSelection()}>
                                                <i className='argo-icon-close' />
                                            </button>
                                            {selectedNode && (
                                                <WorkflowNodeInfo
                                                    node={selectedNode}
                                                    workflow={this.state.workflow}
                                                    onShowContainerLogs={(nodeId, container) => this.openContainerLogsPanel(nodeId, container)}
                                                    onShowYaml={nodeId => this.openNodeYaml(nodeId)}
                                                    archived={false}
                                                />
                                            )}
                                        </div>
                                    </div>
                                ))}
                        </div>
                        {this.state.workflow && (
                            <SlidingPanel isShown={this.selectedNodeId && !!this.sidePanel} onClose={() => this.closeSidePanel()}>
                                {this.sidePanel && this.sidePanel.type === 'logs' && (
                                    <WorkflowLogsViewer workflow={this.state.workflow} nodeId={this.sidePanel.nodeId} container={this.sidePanel.container} archived={false} />
                                )}
                                {this.sidePanel && this.sidePanel.type === 'yaml' && <WorkflowYamlViewer workflow={this.state.workflow} selectedNode={selectedNode} />}
                            </SlidingPanel>
                        )}
                    </Page>
                )}
            </Consumer>
        );
    }

    private deleteWorkflow(ctx: ContextApis) {
        if (!confirm('Are you sure you want to delete this workflow?\nThere is no undo.')) {
            return;
        }
        services.workflows
            .delete(this.props.match.params.name, this.props.match.params.namespace)
            .then(wfDeleteRes => ctx.navigation.goto(uiUrl(`workflows/`)))
            .catch(error => {
                this.appContext.apis.notifications.show({
                    content: 'Unable to delete workflow',
                    type: NotificationType.Error
                });
            });
    }

    private terminateWorkflow(ctx: ContextApis) {
        if (!confirm('Are you sure you want to terminate this workflow?')) {
            return;
        }
        services.workflows
            .terminate(this.props.match.params.name, this.props.match.params.namespace)
            .then(wf => ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
            .catch(error => {
                this.appContext.apis.notifications.show({
                    content: 'Unable to terminate workflow',
                    type: NotificationType.Error
                });
            });
    }

    private resumeWorkflow(ctx: ContextApis) {
        services.workflows
            .resume(this.props.match.params.name, this.props.match.params.namespace)
            .then(wf => ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
            .catch(error => {
                this.appContext.apis.notifications.show({
                    content: 'Unable to resume workflow',
                    type: NotificationType.Error
                });
            });
    }

    private suspendWorkflow(ctx: ContextApis) {
        services.workflows
            .suspend(this.props.match.params.name, this.props.match.params.namespace)
            .then(wf => ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
            .catch(error => {
                this.appContext.apis.notifications.show({
                    content: 'Unable to suspend workflow',
                    type: NotificationType.Error
                });
            });
    }

    private resubmitWorkflow(ctx: ContextApis) {
        if (!confirm('Are you sure you want to re-submit this workflow?')) {
            return;
        }
        services.workflows
            .resubmit(this.props.match.params.name, this.props.match.params.namespace)
            .then(wf => ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
            .catch(error => {
                this.appContext.apis.notifications.show({
                    content: 'Unable to resubmit workflow',
                    type: NotificationType.Error
                });
            });
    }

    private retryWorkflow(ctx: ContextApis) {
        services.workflows
            .retry(this.props.match.params.name, this.props.match.params.namespace)
            .then(wf => ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`)))
            .catch(error => {
                this.appContext.apis.notifications.show({
                    content: 'Unable to retry workflow',
                    type: NotificationType.Error
                });
            });
    }

    private openNodeYaml(nodeId: string) {
        const params = new URLSearchParams(this.appContext.router.route.location.search);
        params.set('sidePanel', `yaml:${nodeId}`);
        this.appContext.router.history.push(`${this.props.match.url}?${params.toString()}`);
    }

    private openContainerLogsPanel(nodeId: string, container: string) {
        const params = new URLSearchParams(this.appContext.router.route.location.search);
        params.set('sidePanel', `logs:${nodeId}:${container}`);
        this.appContext.router.history.push(`${this.props.match.url}?${params.toString()}`);
    }

    private closeSidePanel() {
        const params = new URLSearchParams(this.appContext.router.route.location.search);
        params.delete('sidePanel');
        this.appContext.router.history.push(`${this.props.match.url}?${params.toString()}`);
    }

    private selectTab(tab: string) {
        this.appContext.router.history.push(`${this.props.match.url}?tab=${tab}&nodeId=${this.selectedNodeId}`);
    }

    private selectNode(nodeId: string) {
        this.appContext.router.history.push(`${this.props.match.url}?tab=${this.selectedTabKey}&nodeId=${nodeId}`);
    }

    private removeNodeSelection() {
        const params = new URLSearchParams(this.appContext.router.route.location.search);
        params.delete('nodeId');
        this.appContext.router.history.push(`${this.props.match.url}?${params.toString()}`);
    }

    private renderSummaryTab() {
        if (!this.state.workflow) {
            return <div>Loading...</div>;
        }
        return (
            <div className='argo-container'>
                <div className='workflow-details__content'>
                    <WorkflowSummaryPanel workflow={this.state.workflow} />
                    {this.state.workflow.spec.arguments && this.state.workflow.spec.arguments.parameters && (
                        <React.Fragment>
                            <h6>Parameters</h6>
                            <WorkflowParametersPanel parameters={this.state.workflow.spec.arguments.parameters} />
                        </React.Fragment>
                    )}
                    <h6>Artifacts</h6>
                    <WorkflowArtifacts workflow={this.state.workflow} archived={false} />
                </div>
            </div>
        );
    }

    private ensureUnsubscribed() {
        if (this.changesSubscription) {
            this.changesSubscription.unsubscribe();
        }
        this.changesSubscription = null;
    }

    private async loadWorkflow(namespace: string, name: string) {
        try {
            this.ensureUnsubscribed();
            this.changesSubscription = services.workflows
                .watch({name, namespace})
                .map(changeEvent => changeEvent.object)
                .catch((error, caught) => {
                    return caught;
                })
                .subscribe(workflow => {
                    this.setState({workflow});
                });
        } catch (e) {
            this.appContext.apis.notifications.show({
                content: 'Unable to load workflow',
                type: NotificationType.Error
            });
        }
    }

    private get appContext(): AppContext {
        return this.context as AppContext;
    }
}

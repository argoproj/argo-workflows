import {AppContext, LogsViewer, NotificationType, Page, SlidingPanel} from 'argo-ui';
import * as classNames from 'classnames';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {Observable, Subscription} from 'rxjs';

import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {services} from '../../../shared/services';

import {NODE_PHASE, NodePhase} from '../../../../models';
import {Consumer} from '../../../shared/context';
import {WorkflowArtifacts} from '../workflow-artifacts';
import {WorkflowDag} from '../workflow-dag/workflow-dag';
import {WorkflowNodeInfo} from '../workflow-node-info/workflow-node-info';
import {WorkflowParametersPanel} from '../workflow-parameters-panel';
import {WorkflowSummaryPanel} from '../workflow-summary-panel';
import {WorkflowTimeline} from '../workflow-timeline/workflow-timeline';
import {WorkflowYamlViewer} from '../workflow-yaml-viewer/workflow-yaml-viewer';

require('./workflow-details.scss');

function parseSidePanelParam(param: string) {
    const [type, nodeId, container] = (param || '').split(':');
    if (type === 'logs' || type === 'yaml') {
        return {type, nodeId, container};
    }
    return null;
}

// TODO(simon): most likely extract this to a util file
function isWorkflowSuspended(wf: models.Workflow): boolean {
    if (wf === null || wf.spec === null) {
        return false;
    }
    if (wf.spec.suspend !== undefined && wf.spec.suspend) {
        return true;
    }
    if (wf.status && wf.status.nodes) {
        for (const node of Object.values(wf.status.nodes)) {
            if (node.type === 'Suspend' && node.phase === 'Running') {
                return true;
            }
        }
    }
    return false;
}

function isWorkflowRunning(wf: models.Workflow): boolean {
    if (wf === null || wf.spec === null) {
        return false;
    }
    return wf.status.phase === 'Running';
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

    public componentWillMount() {
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
        const selectedNode = this.state.workflow && this.state.workflow.status && this.state.workflow.status.nodes[this.selectedNodeId];
        const workflowPhase: NodePhase = this.state.workflow && this.state.workflow.status ? this.state.workflow.status.phase : undefined;
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title={'Workflow Details'}
                        toolbar={{
                            breadcrumbs: [{title: 'Workflows', path: uiUrl('workflows')}, {title: this.props.match.params.name}],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Retry',
                                        iconClassName: 'fa fa-undo',
                                        disabled: workflowPhase === undefined || !(workflowPhase === 'Failed' || workflowPhase === 'Error'),
                                        action: () => {
                                            // TODO(simon): most likely extract this somewhere with higher scope
                                            services.workflows
                                                .retry(this.props.match.params.name, this.props.match.params.namespace)
                                                .then(wf => ctx.navigation.goto(`/workflows/${wf.metadata.namespace}/${wf.metadata.name}`))
                                                .catch(error => {
                                                    this.appContext.apis.notifications.show({
                                                        content: 'Unable to retry workflow',
                                                        type: NotificationType.Error
                                                    });
                                                });
                                        }
                                    },
                                    {
                                        title: 'Resubmit',
                                        iconClassName: 'fa fa-plus-circle ',
                                        action: () => {
                                            // TODO(simon): most likely extract this somewhere with higher scope
                                            services.workflows
                                                .resubmit(this.props.match.params.name, this.props.match.params.namespace)
                                                .then(wf => ctx.navigation.goto(`/workflows/${wf.metadata.namespace}/${wf.metadata.name}`))
                                                .catch(error => {
                                                    this.appContext.apis.notifications.show({
                                                        content: 'Unable to resubmit workflow',
                                                        type: NotificationType.Error
                                                    });
                                                });
                                        }
                                    },
                                    {
                                        title: 'Suspend',
                                        iconClassName: 'fa fa-pause',
                                        disabled: isWorkflowRunning(this.state.workflow) && isWorkflowSuspended(this.state.workflow),
                                        action: () => {
                                            // TODO(simon): most likely extract this somewhere with higher scope
                                            services.workflows
                                                .suspend(this.props.match.params.name, this.props.match.params.namespace)
                                                .then(wf => ctx.navigation.goto(`/workflows/${wf.metadata.namespace}/${wf.metadata.name}`))
                                                .catch(error => {
                                                    this.appContext.apis.notifications.show({
                                                        content: 'Unable to suspend workflow',
                                                        type: NotificationType.Error
                                                    });
                                                });
                                        }
                                    },
                                    {
                                        title: 'Resume',
                                        iconClassName: 'fa fa-play',
                                        disabled: !isWorkflowSuspended(this.state.workflow),
                                        action: () => {
                                            // TODO(simon): most likely extract this somewhere with higher scope
                                            services.workflows
                                                .resume(this.props.match.params.name, this.props.match.params.namespace)
                                                .then(wf => ctx.navigation.goto(`/workflows/${wf.metadata.namespace}/${wf.metadata.name}`))
                                                .catch(error => {
                                                    this.appContext.apis.notifications.show({
                                                        content: 'Unable to resume workflow',
                                                        type: NotificationType.Error
                                                    });
                                                });
                                        }
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
                                                />
                                            )}
                                        </div>
                                    </div>
                                ))}
                        </div>
                        {this.state.workflow && (
                            <SlidingPanel isShown={this.selectedNodeId && !!this.sidePanel} onClose={() => this.closeSidePanel()}>
                                {this.sidePanel && this.sidePanel.type === 'logs' && (
                                    <LogsViewer
                                        source={{
                                            key: this.sidePanel.nodeId,
                                            loadLogs: () => services.workflows.getContainerLogs(this.state.workflow, this.sidePanel.nodeId, this.sidePanel.container || 'main'),
                                            shouldRepeat: () => this.state.workflow.status.nodes[this.sidePanel.nodeId].phase === 'Running'
                                        }}
                                    />
                                )}
                                {this.sidePanel && this.sidePanel.type === 'yaml' && <WorkflowYamlViewer workflow={this.state.workflow} selectedNode={selectedNode} />}
                            </SlidingPanel>
                        )}
                    </Page>
                )}
            </Consumer>
        );
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
                    <WorkflowArtifacts workflow={this.state.workflow} />
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
            const workflowUpdates = Observable.from([await services.workflows.get(namespace, name)]).merge(
                services.workflows.watch({name, namespace}).map(changeEvent => changeEvent.object)
            );
            this.changesSubscription = workflowUpdates.subscribe(workflow => {
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

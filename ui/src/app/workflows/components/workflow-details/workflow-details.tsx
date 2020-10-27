import {AppContext, NotificationType, Page, SlidingPanel} from 'argo-ui';
import * as classNames from 'classnames';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {Subscription} from 'rxjs';

import {Link, NodePhase, Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {services} from '../../../shared/services';

import {WorkflowArtifacts, WorkflowLogsViewer, WorkflowNodeInfo, WorkflowPanel, WorkflowSummaryPanel, WorkflowTimeline, WorkflowYamlViewer} from '..';
import {CostOptimisationNudge} from '../../../shared/components/cost-optimisation-nudge';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {hasWarningConditionBadge} from '../../../shared/conditions-panel';
import {Consumer, ContextApis} from '../../../shared/context';
import * as Operations from '../../../shared/workflow-operations-map';
import {WorkflowOperationAction, WorkflowOperationName, WorkflowOperations} from '../../../shared/workflow-operations-map';
import {EventsPanel} from '../events-panel';
import {WorkflowParametersPanel} from '../workflow-parameters-panel';
import {WorkflowResourcePanel} from './workflow-resource-panel';

require('./workflow-details.scss');

function parseSidePanelParam(param: string) {
    const [type, nodeId, container] = (param || '').split(':');
    if (type === 'logs' || type === 'yaml') {
        return {type, nodeId, container: container || 'main'};
    }
    return null;
}

interface WorkflowDetailsState {
    workflow?: Workflow;
    links?: Link[];
    error?: Error;
}

export class WorkflowDetails extends React.Component<RouteComponentProps<any>, WorkflowDetailsState> {
    public static contextTypes = {
        router: PropTypes.object,
        apis: PropTypes.object
    };

    private changesSubscription: Subscription;
    private timelineComponent: WorkflowTimeline;

    private get resourceVersion() {
        return this.state.workflow && this.state.workflow.metadata.resourceVersion;
    }

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
        this.state = {};
    }

    public componentDidMount() {
        this.loadWorkflow(this.props.match.params.namespace, this.props.match.params.name);
        services.info.getInfo().then(info => this.setState({links: info.links}));
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
                                {title: this.props.match.params.namespace + '/' + this.props.match.params.name}
                            ],
                            actionMenu: {
                                items: this.getItems(workflowPhase, ctx)
                            },
                            tools: (
                                <div className='workflow-details__topbar-buttons'>
                                    <a className={classNames({active: this.selectedTabKey === 'summary'})} onClick={() => this.selectTab('summary')}>
                                        <i className='fa fa-columns' />
                                        {this.state.workflow && this.state.workflow.status.conditions && hasWarningConditionBadge(this.state.workflow.status.conditions) && (
                                            <span className='badge' />
                                        )}
                                    </a>
                                    <a className={classNames({active: this.selectedTabKey === 'events'})} onClick={() => this.selectTab('events')}>
                                        <i className='fa argo-icon-notification' />
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
                                                <WorkflowPanel
                                                    workflowMetadata={this.state.workflow.metadata}
                                                    workflowStatus={this.state.workflow.status}
                                                    selectedNodeId={this.selectedNodeId}
                                                    nodeClicked={nodeId => this.selectNode(nodeId)}
                                                />
                                            )) ||
                                                (this.selectedTabKey === 'events' && (
                                                    <EventsPanel namespace={this.state.workflow.metadata.namespace} kind='Workflow' name={this.state.workflow.metadata.name} />
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
                                                    links={this.state.links}
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

    private performAction(action: WorkflowOperationAction, title: WorkflowOperationName, ctx: ContextApis): void {
        if (!confirm(`Are you sure you want to ${title.toLowerCase()} this workflow?`)) {
            return;
        }
        action(this.state.workflow)
            .then(wf => {
                if (title === 'DELETE') {
                    ctx.navigation.goto(uiUrl(``));
                } else {
                    ctx.navigation.goto(uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`));
                }
            })
            .catch(() => {
                this.appContext.apis.notifications.show({
                    content: `Unable to ${title} workflow`,
                    type: NotificationType.Error
                });
            });
    }

    private getItems(workflowPhase: NodePhase, ctx: any) {
        const workflowOperationsMap: WorkflowOperations = Operations.WorkflowOperationsMap;
        const items = Object.keys(workflowOperationsMap).map(actionName => {
            const workflowOperation = workflowOperationsMap[actionName];
            return {
                title: workflowOperation.title.charAt(0).toUpperCase() + workflowOperation.title.slice(1),
                iconClassName: workflowOperation.iconClassName,
                disabled: workflowOperation.disabled(this.state.workflow),
                action: () => this.performAction(workflowOperation.action, workflowOperation.title, ctx)
            };
        });

        if (this.state.links) {
            this.state.links
                .filter(link => link.scope === 'workflow')
                .forEach(link => {
                    items.push({
                        title: link.name,
                        iconClassName: 'fa fa-link',
                        disabled: false,
                        action: () => this.openLink(link)
                    });
                });
        }
        return items;
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

    private renderCostOptimisations() {
        const recommendations: string[] = [];
        if (!this.state.workflow.spec.activeDeadlineSeconds) {
            recommendations.push('activeDeadlineSeconds');
        }
        if (!this.state.workflow.spec.ttlStrategy) {
            recommendations.push('ttlStrategy');
        }
        if (!this.state.workflow.spec.podGC) {
            recommendations.push('podGC');
        }
        if (recommendations.length === 0) {
            return;
        }
        return (
            <CostOptimisationNudge name='workflow'>
                You do not have {recommendations.join('/')} enabled for this workflow. Enabling these will reduce your costs.
            </CostOptimisationNudge>
        );
    }

    private renderSummaryTab() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} style={{margin: 20}} />;
        }
        if (!this.state.workflow) {
            return <Loading />;
        }
        return (
            <div className='argo-container'>
                <div className='workflow-details__content'>
                    <WorkflowSummaryPanel workflow={this.state.workflow} />
                    {this.renderCostOptimisations()}
                    {this.state.workflow.spec.arguments && this.state.workflow.spec.arguments.parameters && (
                        <React.Fragment>
                            <h6>Parameters</h6>
                            <WorkflowParametersPanel parameters={this.state.workflow.spec.arguments.parameters} />
                        </React.Fragment>
                    )}
                    <h6>Artifacts</h6>
                    <WorkflowArtifacts workflow={this.state.workflow} archived={false} />
                    <WorkflowResourcePanel workflow={this.state.workflow} />
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

    private loadWorkflow(namespace: string, name: string) {
        try {
            this.ensureUnsubscribed();
            this.changesSubscription = services.workflows
                .watch({name, namespace, resourceVersion: this.resourceVersion})
                .map(changeEvent => changeEvent.object)
                .subscribe(
                    workflow => this.setState({workflow, error: null}),
                    error => this.setState({error}, () => this.loadWorkflow(namespace, name))
                );
        } catch (error) {
            this.setState({error});
        }
    }

    private get appContext(): AppContext {
        return this.context as AppContext;
    }

    private openLink(link: Link) {
        const url = link.url.replace('${metadata.namespace}', this.state.workflow.metadata.namespace).replace('${metadata.name}', this.state.workflow.metadata.name);
        if ((window.event as MouseEvent).ctrlKey) {
            window.open(url, '_blank');
        } else {
            document.location.href = url;
        }
    }
}

import {NotificationType, Page, SlidingPanel} from 'argo-ui';
import * as classNames from 'classnames';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {Link, Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {ResourceEditor} from '../../../shared/components/resource-editor/resource-editor';
import {services} from '../../../shared/services';
import {
    WorkflowArtifacts,
    WorkflowLogsViewer,
    WorkflowNodeInfo,
    WorkflowPanel,
    WorkflowParametersPanel,
    WorkflowSummaryPanel,
    WorkflowTimeline,
    WorkflowYamlViewer
} from '../../../workflows/components';
import {WorkflowResourcePanel} from '../../../workflows/components/workflow-details/workflow-resource-panel';

require('../../../workflows/components/workflow-details/workflow-details.scss');

interface State {
    workflow?: Workflow;
    links?: Link[];
    error?: Error;
}

export class ArchivedWorkflowDetails extends BasePage<RouteComponentProps<any>, State> {
    private get uid() {
        return this.props.match.params.uid;
    }

    private get tab() {
        return this.queryParam('tab') || 'workflow';
    }

    private set tab(tab) {
        this.setQueryParams({tab});
    }

    private get nodeId() {
        return this.queryParam('nodeId');
    }

    private set nodeId(nodeId) {
        this.setQueryParams({nodeId});
    }

    private get container() {
        return this.queryParam('container') || 'main';
    }

    private get sidePanel() {
        return this.queryParam('sidePanel');
    }

    private set sidePanel(sidePanel) {
        this.setQueryParams({sidePanel});
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {};
    }

    public componentDidMount(): void {
        services.info
            .getInfo()
            .then(info => this.setState({links: info.links}))
            .then(() => services.archivedWorkflows.get(this.uid).then(workflow => this.setState({error: null, workflow})))
            .catch(error => this.setState({error}));
    }

    public render() {
        const items = [
            {
                title: 'Resubmit',
                iconClassName: 'fa fa-redo',
                action: () => (this.sidePanel = 'resubmit')
            },
            {
                title: 'Delete',
                iconClassName: 'fa fa-trash',
                action: () => this.deleteArchivedWorkflow()
            }
        ];
        if (this.state.links) {
            this.state.links
                .filter(link => link.scope === 'workflow')
                .forEach(link =>
                    items.push({
                        title: link.name,
                        iconClassName: 'fa fa-link',
                        action: () => this.openLink(link)
                    })
                );
        }
        return (
            <Page
                title='Archived Workflow Details'
                toolbar={{
                    actionMenu: {
                        items
                    },
                    breadcrumbs: [
                        {
                            title: 'Archived Workflows',
                            path: uiUrl('archived-workflows/')
                        },
                        {title: this.uid}
                    ],
                    tools: (
                        <div className='workflow-details__topbar-buttons'>
                            <a className={classNames({active: this.tab === 'summary'})} onClick={() => (this.tab = 'summary')}>
                                <i className='fa fa-columns' />
                            </a>
                            <a className={classNames({active: this.tab === 'timeline'})} onClick={() => (this.tab = 'timeline')}>
                                <i className='fa argo-icon-timeline' />
                            </a>
                            <a className={classNames({active: this.tab === 'workflow'})} onClick={() => (this.tab = 'workflow')}>
                                <i className='fa argo-icon-workflow' />
                            </a>
                        </div>
                    )
                }}>
                <div className={classNames('workflow-details', {'workflow-details--step-node-expanded': !!this.nodeId})}>{this.renderArchivedWorkflowDetails()}</div>
            </Page>
        );
    }

    private renderArchivedWorkflowDetails() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} style={{margin: 20}} />;
        }
        if (!this.state.workflow) {
            return <Loading />;
        }
        return (
            <>
                {this.tab === 'summary' ? (
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
                            <WorkflowArtifacts workflow={this.state.workflow} archived={true} />
                            <WorkflowResourcePanel workflow={this.state.workflow} />
                        </div>
                    </div>
                ) : (
                    <div>
                        <div className='workflow-details__graph-container'>
                            {this.tab === 'workflow' ? (
                                <WorkflowPanel
                                    workflowMetadata={this.state.workflow.metadata}
                                    workflowStatus={this.state.workflow.status}
                                    selectedNodeId={this.nodeId}
                                    nodeClicked={nodeId => (this.nodeId = nodeId)}
                                />
                            ) : (
                                <WorkflowTimeline workflow={this.state.workflow} selectedNodeId={this.nodeId} nodeClicked={node => (this.nodeId = node.id)} />
                            )}
                        </div>
                        {this.nodeId && (
                            <div className='workflow-details__step-info'>
                                <button className='workflow-details__step-info-close' onClick={() => (this.nodeId = null)}>
                                    <i className='argo-icon-close' />
                                </button>
                                <WorkflowNodeInfo
                                    node={this.node}
                                    workflow={this.state.workflow}
                                    links={this.state.links}
                                    onShowYaml={nodeId =>
                                        this.setQueryParams({
                                            sidePanel: 'yaml',
                                            nodeId
                                        })
                                    }
                                    onShowContainerLogs={(nodeId, container) =>
                                        this.setQueryParams({
                                            sidePanel: 'logs',
                                            nodeId,
                                            container
                                        })
                                    }
                                    archived={true}
                                />
                            </div>
                        )}
                    </div>
                )}
                <SlidingPanel isShown={!!this.sidePanel} onClose={() => (this.sidePanel = null)}>
                    {this.sidePanel === 'yaml' && <WorkflowYamlViewer workflow={this.state.workflow} selectedNode={this.node} />}
                    {this.sidePanel === 'logs' && <WorkflowLogsViewer workflow={this.state.workflow} nodeId={this.nodeId} container={this.container} archived={true} />}
                    {this.sidePanel === 'resubmit' && (
                        <ResourceEditor<Workflow>
                            editing={true}
                            title='Resubmit Archived Workflow'
                            kind='Workflow'
                            value={{
                                metadata: {
                                    namespace: this.state.workflow.metadata.namespace,
                                    name: this.state.workflow.metadata.name
                                },
                                spec: this.state.workflow.spec
                            }}
                            onSubmit={(value: Workflow) =>
                                services.workflows
                                    .create(value, value.metadata.namespace)
                                    .then(workflow => (document.location.href = uiUrl(`workflows/${workflow.metadata.namespace}/${workflow.metadata.name}`)))
                            }
                        />
                    )}
                </SlidingPanel>
            </>
        );
    }

    private get node() {
        return this.nodeId && this.state.workflow.status.nodes[this.nodeId];
    }

    private deleteArchivedWorkflow() {
        if (!confirm('Are you sure you want to delete this archived workflow?\nThere is no undo.')) {
            return;
        }
        services.archivedWorkflows
            .delete(this.uid)
            .then(() => {
                document.location.href = uiUrl('archived-workflows');
            })
            .catch(e => {
                this.appContext.apis.notifications.show({
                    content: 'Failed to delete archived workflow ' + e,
                    type: NotificationType.Error
                });
            });
    }

    private openLink(link: Link) {
        document.location.href = link.url
            .replace(/\${metadata\.namespace}/g, this.state.workflow.metadata.namespace)
            .replace(/\${metadata\.name}/g, this.state.workflow.metadata.name)
            .replace(/\${status\.startedAt}/g, this.state.workflow.status.startedAt)
            .replace(/\${status\.finishedAt}/g, this.state.workflow.status.finishedAt);
    }
}

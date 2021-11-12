import {NotificationType, Page, SlidingPanel} from 'argo-ui';
import * as classNames from 'classnames';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {execSpec, Link, Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {ProcessURL} from '../../../shared/components/links';
import {Loading} from '../../../shared/components/loading';
import {ResourceEditor} from '../../../shared/components/resource-editor/resource-editor';
import {services} from '../../../shared/services';
import {WorkflowArtifacts} from '../../../workflows/components/workflow-artifacts';

import {ANNOTATION_KEY_POD_NAME_VERSION} from '../../../shared/annotations';
import {getPodName, getTemplateNameFromNode} from '../../../shared/pod-name';
import {WorkflowResourcePanel} from '../../../workflows/components/workflow-details/workflow-resource-panel';
import {WorkflowLogsViewer} from '../../../workflows/components/workflow-logs-viewer/workflow-logs-viewer';
import {WorkflowNodeInfo} from '../../../workflows/components/workflow-node-info/workflow-node-info';
import {WorkflowPanel} from '../../../workflows/components/workflow-panel/workflow-panel';
import {WorkflowParametersPanel} from '../../../workflows/components/workflow-parameters-panel';
import {WorkflowSummaryPanel} from '../../../workflows/components/workflow-summary-panel';
import {WorkflowTimeline} from '../../../workflows/components/workflow-timeline/workflow-timeline';
import {WorkflowYamlViewer} from '../../../workflows/components/workflow-yaml-viewer/workflow-yaml-viewer';

require('../../../workflows/components/workflow-details/workflow-details.scss');

interface State {
    workflow?: Workflow;
    links?: Link[];
    error?: Error;
}

export class ArchivedWorkflowDetails extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.props.match.params.namespace;
    }

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
            .then(() =>
                services.archivedWorkflows.get(this.uid).then(workflow =>
                    this.setState({
                        error: null,
                        workflow
                    })
                )
            )
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
                        iconClassName: 'fa fa-external-link-alt',
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
                        {title: 'Archived Workflows', path: uiUrl('archived-workflows')},
                        {
                            title: this.namespace,
                            path: uiUrl('archived-workflows/' + this.namespace)
                        },
                        {
                            title: this.uid,
                            path: uiUrl('archived-workflows/' + this.namespace + '/' + this.uid)
                        }
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
            return <ErrorNotice error={this.state.error} />;
        }
        if (!this.state.workflow) {
            return <Loading />;
        }
        return (
            <>
                {this.tab === 'summary' ? (
                    <div className='workflow-details__container'>
                        <div className='argo-container'>
                            <div className='workflow-details__content'>
                                <WorkflowSummaryPanel workflow={this.state.workflow} />
                                {execSpec(this.state.workflow).arguments && execSpec(this.state.workflow).arguments.parameters && (
                                    <React.Fragment>
                                        <h6>Parameters</h6>
                                        <WorkflowParametersPanel parameters={execSpec(this.state.workflow).arguments.parameters} />
                                    </React.Fragment>
                                )}
                                <h6>Artifacts</h6>
                                <WorkflowArtifacts workflow={this.state.workflow} archived={true} />
                                <WorkflowResourcePanel workflow={this.state.workflow} />
                            </div>
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
                    {this.sidePanel === 'logs' && (
                        <WorkflowLogsViewer workflow={this.state.workflow} initialPodName={this.podName} nodeId={this.nodeId} container={this.container} archived={true} />
                    )}
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

    private get podName() {
        if (this.nodeId && this.state.workflow) {
            const workflowName = this.state.workflow.metadata.name;
            let annotations: {[name: string]: string} = {};
            if (typeof this.state.workflow.metadata.annotations !== 'undefined') {
                annotations = this.state.workflow.metadata.annotations;
            }
            const version = annotations[ANNOTATION_KEY_POD_NAME_VERSION];
            const templateName = getTemplateNameFromNode(this.node);
            return getPodName(workflowName, this.node.name, templateName, this.nodeId, version);
        }
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
        const object = {
            metadata: {
                namespace: this.state.workflow.metadata.namespace,
                name: this.state.workflow.metadata.name
            },
            workflow: this.state.workflow,
            status: {
                startedAt: this.state.workflow.status.startedAt,
                finishedAt: this.state.workflow.status.finishedAt
            }
        };
        const url = ProcessURL(link.url, object);

        if ((window.event as MouseEvent).ctrlKey || (window.event as MouseEvent).metaKey) {
            window.open(url, '_blank');
        } else {
            document.location.href = url;
        }
    }
}

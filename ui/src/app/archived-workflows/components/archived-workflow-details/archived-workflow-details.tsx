import {NotificationType, Page, SlidingPanel} from 'argo-ui';
import * as classNames from 'classnames';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {ArtifactRepository, execSpec, Link, Workflow} from '../../../../models';
import {artifactRepoHasLocation, findArtifact} from '../../../shared/artifacts';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {ProcessURL} from '../../../shared/components/links';
import {Loading} from '../../../shared/components/loading';
import {Context} from '../../../shared/context';
import {services} from '../../../shared/services';
import {useQueryParams} from '../../../shared/use-query-params';
import {WorkflowArtifacts} from '../../../workflows/components/workflow-artifacts';

import {ANNOTATION_KEY_POD_NAME_VERSION} from '../../../shared/annotations';
import {getPodName, getTemplateNameFromNode} from '../../../shared/pod-name';
import {getResolvedTemplates} from '../../../shared/template-resolution';
import {ArtifactPanel} from '../../../workflows/components/workflow-details/artifact-panel';
import {WorkflowResourcePanel} from '../../../workflows/components/workflow-details/workflow-resource-panel';
import {WorkflowLogsViewer} from '../../../workflows/components/workflow-logs-viewer/workflow-logs-viewer';
import {WorkflowNodeInfo} from '../../../workflows/components/workflow-node-info/workflow-node-info';
import {WorkflowPanel} from '../../../workflows/components/workflow-panel/workflow-panel';
import {WorkflowParametersPanel} from '../../../workflows/components/workflow-parameters-panel';
import {WorkflowSummaryPanel} from '../../../workflows/components/workflow-summary-panel';
import {WorkflowTimeline} from '../../../workflows/components/workflow-timeline/workflow-timeline';
import {WorkflowYamlViewer} from '../../../workflows/components/workflow-yaml-viewer/workflow-yaml-viewer';

require('../../../workflows/components/workflow-details/workflow-details.scss');

const STEP_GRAPH_CONTAINER_MIN_WIDTH = 490;
const STEP_INFO_WIDTH = 570;

export const ArchivedWorkflowDetails = ({history, location, match}: RouteComponentProps<any>) => {
    const ctx = useContext(Context);
    const queryParams = new URLSearchParams(location.search);
    const [workflow, setWorkflow] = useState<Workflow>();
    const [links, setLinks] = useState<Link[]>();
    const [error, setError] = useState<Error>();

    const [namespace] = useState(match.params.namespace);
    const [uid] = useState(match.params.uid);
    const [tab, setTab] = useState(queryParams.get('tab') || 'workflow');
    const [nodeId, setNodeId] = useState(queryParams.get('nodeId'));
    const [container, setContainer] = useState(queryParams.get('container') || 'main');
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel'));
    const selectedArtifact = workflow && workflow.status && findArtifact(workflow.status, nodeId);
    const [selectedTemplateArtifactRepo, setSelectedTemplateArtifactRepo] = useState<ArtifactRepository>();
    const node = nodeId && workflow.status.nodes[nodeId];

    const podName = () => {
        if (nodeId && workflow) {
            const workflowName = workflow.metadata.name;
            const annotations = workflow.metadata.annotations || {};
            const version = annotations[ANNOTATION_KEY_POD_NAME_VERSION];
            const templateName = getTemplateNameFromNode(node);
            return getPodName(workflowName, node.name, templateName, nodeId, version);
        }
    };

    useEffect(
        useQueryParams(history, p => {
            setSidePanel(p.get('sidePanel'));
            setNodeId(p.get('nodeId'));
            setContainer(p.get('container'));
        }),
        [history]
    );

    useEffect(() => {
        services.info
            .getInfo()
            .then(info => setLinks(info.links))
            .then(() =>
                services.archivedWorkflows.get(uid, namespace).then(retrievedWorkflow => {
                    setError(null);
                    setWorkflow(retrievedWorkflow);
                })
            )
            .catch(newError => setError(newError));
        services.info.collectEvent('openedArchivedWorkflowDetails').then();
    }, []);

    useEffect(() => {
        // update the default Artifact Repository for the Template that corresponds to the selectedArtifact
        // if there's an ArtifactLocation configured for the Template we use that
        // otherwise we use the central one for the Workflow configured in workflow.status.artifactRepositoryRef.artifactRepository
        // (Note that individual Artifacts may also override whatever this gets set to)
        if (workflow?.status?.nodes && selectedArtifact) {
            const template = getResolvedTemplates(workflow, workflow.status.nodes[selectedArtifact.nodeId]);
            const artifactRepo = template?.archiveLocation;
            if (artifactRepo && artifactRepoHasLocation(artifactRepo)) {
                setSelectedTemplateArtifactRepo(artifactRepo);
            } else {
                setSelectedTemplateArtifactRepo(workflow.status.artifactRepositoryRef.artifactRepository);
            }
        }
    }, [workflow, selectedArtifact]);

    const renderArchivedWorkflowDetails = () => {
        if (error) {
            return <ErrorNotice error={error} />;
        }
        if (!workflow) {
            return <Loading />;
        }
        return (
            <>
                {tab === 'summary' ? (
                    <div className='workflow-details__container'>
                        <div className='argo-container'>
                            <div className='workflow-details__content'>
                                <WorkflowSummaryPanel workflow={workflow} />
                                {execSpec(workflow).arguments && execSpec(workflow).arguments.parameters && (
                                    <React.Fragment>
                                        <h6>Parameters</h6>
                                        <WorkflowParametersPanel parameters={execSpec(workflow).arguments.parameters} />
                                    </React.Fragment>
                                )}
                                <h6>Artifacts</h6>
                                <WorkflowArtifacts workflow={workflow} archived={true} />
                                <WorkflowResourcePanel workflow={workflow} />
                            </div>
                        </div>
                    </div>
                ) : (
                    <div className='workflow-details__graph-container-wrapper'>
                        <div className='workflow-details__graph-container' style={{minWidth: STEP_GRAPH_CONTAINER_MIN_WIDTH, width: '100%'}}>
                            {tab === 'workflow' ? (
                                <WorkflowPanel
                                    workflowMetadata={workflow.metadata}
                                    workflowStatus={workflow.status}
                                    selectedNodeId={nodeId}
                                    nodeClicked={newNodeId => setNodeId(newNodeId)}
                                />
                            ) : (
                                <WorkflowTimeline workflow={workflow} selectedNodeId={nodeId} nodeClicked={newNode => setNodeId(newNode.id)} />
                            )}
                        </div>
                        {nodeId && (
                            <div className='workflow-details__step-info' style={{width: STEP_INFO_WIDTH, float: 'right'}}>
                                <button className='workflow-details__step-info-close' onClick={() => setNodeId(null)}>
                                    <i className='argo-icon-close' />
                                </button>
                                {node && (
                                    <WorkflowNodeInfo
                                        node={node}
                                        workflow={workflow}
                                        links={links}
                                        onShowYaml={newNodeId => {
                                            setSidePanel('yaml');
                                            setNodeId(newNodeId);
                                        }}
                                        onShowContainerLogs={(newNodeId, newContainer) => {
                                            setSidePanel('logs');
                                            setNodeId(newNodeId);
                                            setContainer(newContainer);
                                        }}
                                        archived={true}
                                    />
                                )}
                                {selectedArtifact && (
                                    <ArtifactPanel workflow={workflow} artifact={selectedArtifact} archived={true} artifactRepository={selectedTemplateArtifactRepo} />
                                )}
                            </div>
                        )}
                    </div>
                )}
                <SlidingPanel isShown={!!sidePanel} onClose={() => setSidePanel(null)}>
                    {sidePanel === 'yaml' && <WorkflowYamlViewer workflow={workflow} selectedNode={node} />}
                    {sidePanel === 'logs' && <WorkflowLogsViewer workflow={workflow} initialPodName={podName()} nodeId={nodeId} container={container} archived={true} />}
                </SlidingPanel>
            </>
        );
    };

    const deleteArchivedWorkflow = () => {
        if (!confirm('Are you sure you want to delete this archived workflow?\nThere is no undo.')) {
            return;
        }
        services.archivedWorkflows
            .delete(uid, workflow.metadata.namespace)
            .then(() => {
                document.location.href = uiUrl('archived-workflows');
            })
            .catch(e => {
                ctx.notifications.show({
                    content: 'Failed to delete archived workflow ' + e,
                    type: NotificationType.Error
                });
            });
    };

    const resubmitArchivedWorkflow = () => {
        if (!confirm('Are you sure you want to resubmit this archived workflow?')) {
            return;
        }
        services.archivedWorkflows
            .resubmit(workflow.metadata.uid, workflow.metadata.namespace)
            .then(newWorkflow => (document.location.href = uiUrl(`workflows/${newWorkflow.metadata.namespace}/${newWorkflow.metadata.name}`)))
            .catch(e => {
                ctx.notifications.show({
                    content: 'Failed to resubmit archived workflow ' + e,
                    type: NotificationType.Error
                });
            });
    };

    const retryArchivedWorkflow = () => {
        if (!confirm('Are you sure you want to retry this archived workflow?')) {
            return;
        }
        services.archivedWorkflows
            .retry(workflow.metadata.uid, workflow.metadata.namespace)
            .then(newWorkflow => (document.location.href = uiUrl(`workflows/${newWorkflow.metadata.namespace}/${newWorkflow.metadata.name}`)))
            .catch(e => {
                ctx.notifications.show({
                    content: 'Failed to retry archived workflow ' + e,
                    type: NotificationType.Error
                });
            });
    };

    const openLink = (link: Link) => {
        const object = {
            metadata: {
                namespace: workflow.metadata.namespace,
                name: workflow.metadata.name
            },
            workflow,
            status: {
                startedAt: workflow.status.startedAt,
                finishedAt: workflow.status.finishedAt
            }
        };
        const url = ProcessURL(link.url, object);

        if ((window.event as MouseEvent).ctrlKey || (window.event as MouseEvent).metaKey) {
            window.open(url, '_blank');
        } else {
            document.location.href = url;
        }
    };

    const workflowPhase = workflow?.status?.phase;
    const items = [
        {
            title: 'Retry',
            iconClassName: 'fa fa-undo',
            disabled: workflowPhase === undefined || !(workflowPhase === 'Failed' || workflowPhase === 'Error'),
            action: () => retryArchivedWorkflow()
        },
        {
            title: 'Resubmit',
            iconClassName: 'fa fa-plus-circle',
            disabled: false,
            action: () => resubmitArchivedWorkflow()
        },
        {
            title: 'Delete',
            iconClassName: 'fa fa-trash',
            disabled: false,
            action: () => deleteArchivedWorkflow()
        }
    ];
    if (links) {
        links
            .filter(link => link.scope === 'workflow')
            .forEach(link =>
                items.push({
                    title: link.name,
                    iconClassName: 'fa fa-external-link-alt',
                    disabled: false,
                    action: () => openLink(link)
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
                        title: namespace,
                        path: uiUrl('archived-workflows/' + namespace)
                    },
                    {
                        title: uid,
                        path: uiUrl('archived-workflows/' + namespace + '/' + uid)
                    }
                ],
                tools: (
                    <div className='workflow-details__topbar-buttons'>
                        <a className={classNames({active: tab === 'summary'})} onClick={() => setTab('summary')}>
                            <i className='fa fa-columns' />
                        </a>
                        <a className={classNames({active: tab === 'timeline'})} onClick={() => setTab('timeline')}>
                            <i className='fa argo-icon-timeline' />
                        </a>
                        <a className={classNames({active: tab === 'workflow'})} onClick={() => setTab('workflow')}>
                            <i className='fa argo-icon-workflow' />
                        </a>
                    </div>
                )
            }}>
            <div className={classNames('workflow-details', {'workflow-details--step-node-expanded': !!nodeId})}>{renderArchivedWorkflowDetails()}</div>
        </Page>
    );
};

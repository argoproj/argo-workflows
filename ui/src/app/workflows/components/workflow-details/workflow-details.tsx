import {Page, SlidingPanel} from 'argo-ui';
import * as classNames from 'classnames';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {execSpec, Link, NodeStatus, Parameter, Workflow} from '../../../../models';
import {ANNOTATION_KEY_POD_NAME_VERSION} from '../../../shared/annotations';
import {findArtifact} from '../../../shared/artifacts';
import {uiUrl} from '../../../shared/base';
import {CostOptimisationNudge} from '../../../shared/components/cost-optimisation-nudge';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {ProcessURL} from '../../../shared/components/links';
import {Loading} from '../../../shared/components/loading';
import {SecurityNudge} from '../../../shared/components/security-nudge';
import {hasArtifactGCError, hasWarningConditionBadge} from '../../../shared/conditions-panel';
import {Context} from '../../../shared/context';
import {historyUrl} from '../../../shared/history';
import {getPodName, getTemplateNameFromNode} from '../../../shared/pod-name';
import {RetryWatch} from '../../../shared/retry-watch';
import {services} from '../../../shared/services';
import {useQueryParams} from '../../../shared/use-query-params';
import * as Operations from '../../../shared/workflow-operations-map';
import {WorkflowOperations} from '../../../shared/workflow-operations-map';
import {WidgetGallery} from '../../../widgets/widget-gallery';
import {EventsPanel} from '../events-panel';
import {WorkflowArtifacts} from '../workflow-artifacts';
import {WorkflowLogsViewer} from '../workflow-logs-viewer/workflow-logs-viewer';
import {WorkflowNodeInfo} from '../workflow-node-info/workflow-node-info';
import {WorkflowPanel} from '../workflow-panel/workflow-panel';
import {WorkflowParametersPanel} from '../workflow-parameters-panel';
import {WorkflowSummaryPanel} from '../workflow-summary-panel';
import {WorkflowTimeline} from '../workflow-timeline/workflow-timeline';
import {WorkflowYamlViewer} from '../workflow-yaml-viewer/workflow-yaml-viewer';
import {ArtifactPanel} from './artifact-panel';
import {SuspendInputs} from './suspend-inputs';
import {WorkflowResourcePanel} from './workflow-resource-panel';

require('./workflow-details.scss');

function parseSidePanelParam(param: string) {
    const [type, nodeId, container] = (param || '').split(':');
    return {type, nodeId, container: container || 'main'};
}

export const WorkflowDetails = ({history, location, match}: RouteComponentProps<any>) => {
    // boiler-plate
    const {navigation, popup} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    const [namespace] = useState(match.params.namespace);
    const [name, setName] = useState(match.params.name);
    const [tab, setTab] = useState(queryParams.get('tab') || 'workflow');
    const [nodeId, setNodeId] = useState(queryParams.get('nodeId'));
    const [nodePanelView, setNodePanelView] = useState(queryParams.get('nodePanelView'));
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel'));
    const [parameters, setParameters] = useState<Parameter[]>([]);

    useEffect(
        useQueryParams(history, p => {
            setTab(p.get('tab') || 'workflow');
            setNodeId(p.get('nodeId'));
            setNodePanelView(p.get('nodePanelView'));
            setSidePanel(p.get('sidePanel'));
        }),
        [history]
    );

    const getInputParametersForNode = (selectedWorkflowNodeId: string): Parameter[] => {
        const selectedWorkflowNode = workflow && workflow.status && workflow.status.nodes && workflow.status.nodes[selectedWorkflowNodeId];
        return (
            selectedWorkflowNode?.inputs?.parameters?.map(param => {
                const paramClone = {...param};
                if (paramClone.enum) {
                    paramClone.value = paramClone.default;
                }
                return paramClone;
            }) || []
        );
    };

    useEffect(() => {
        history.push(historyUrl('workflows/{namespace}/{name}', {namespace, name, tab, nodeId, nodePanelView, sidePanel}));
    }, [namespace, name, tab, nodeId, nodePanelView, sidePanel]);

    const [workflow, setWorkflow] = useState<Workflow>();
    const [links, setLinks] = useState<Link[]>();
    const [error, setError] = useState<Error>();
    useEffect(() => {
        services.info
            .getInfo()
            .then(info => setLinks(info.links))
            .catch(setError);
        services.info.collectEvent('openedWorkflowDetails').then();
    }, []);

    useEffect(() => {
        setParameters(getInputParametersForNode(nodeId));
    }, [nodeId, workflow]);

    const parsedSidePanel = parseSidePanelParam(sidePanel);

    const getItems = () => {
        const workflowOperationsMap: WorkflowOperations = Operations.WorkflowOperationsMap;
        const items = Object.keys(workflowOperationsMap)
            .filter(actionName => !workflowOperationsMap[actionName].disabled(workflow))
            .map(actionName => {
                const workflowOperation = workflowOperationsMap[actionName];
                return {
                    title: workflowOperation.title.charAt(0).toUpperCase() + workflowOperation.title.slice(1),
                    iconClassName: workflowOperation.iconClassName,
                    action: () => {
                        popup.confirm('Confirm', `Are you sure you want to ${workflowOperation.title.toLowerCase()} this workflow?`).then(yes => {
                            if (yes) {
                                workflowOperation
                                    .action(workflow)
                                    .then((wf: Workflow) => {
                                        if (workflowOperation.title === 'DELETE') {
                                            navigation.goto(uiUrl(`workflows/${workflow.metadata.namespace}`));
                                        } else {
                                            setName(wf.metadata.name);
                                        }
                                    })
                                    .catch(setError);
                            }
                        });
                    }
                };
            });

        items.push({
            action: () => setSidePanel('logs'),
            iconClassName: 'fa fa-bars',
            title: 'Logs'
        });

        items.push({
            action: () => setSidePanel('share'),
            iconClassName: 'fa fa-share-alt',
            title: 'Share'
        });

        if (links) {
            links
                .filter(link => link.scope === 'workflow')
                .forEach(link => {
                    items.push({
                        title: link.name,
                        iconClassName: 'fa fa-external-link-alt',
                        action: () => openLink(link)
                    });
                });
        }

        // we only want one link, and we have a preference
        for (const k of [
            'workflows.argoproj.io/workflow-template',
            'workflows.argoproj.io/cluster-workflow-template',
            'workflows.argoproj.io/cron-workflow',
            'workflows.argoproj.io/workflow-event-binding',
            'workflows.argoproj.io/resubmitted-from-workflow'
        ]) {
            const v = workflow?.metadata.labels[k];
            if (v) {
                items.push({
                    title: 'Previous Runs',
                    iconClassName: 'fa fa-search',
                    action: () => navigation.goto(uiUrl(`workflows/${workflow.metadata.namespace}?label=${k}=${v}`))
                });
                break; // only add one item
            }
        }

        if (workflow?.spec?.workflowTemplateRef) {
            const templateName: string = workflow.spec.workflowTemplateRef.name;
            const clusterScope: boolean = workflow.spec.workflowTemplateRef.clusterScope;
            const url: string = clusterScope ? `/cluster-workflow-templates/${templateName}` : `/workflow-templates/${workflow.metadata.namespace}/${templateName}`;
            const icon: string = clusterScope ? 'fa fa-window-restore' : 'fa fa-window-maximize';

            const templateLink: Link = {
                name: 'Open Workflow Template',
                scope: 'workflow',
                url
            };

            items.push({
                title: templateLink.name,
                iconClassName: icon,
                action: () => openLink(templateLink)
            });
        }

        return items;
    };

    const renderSecurityNudge = () => {
        if (!execSpec(workflow).securityContext) {
            return <SecurityNudge>This workflow does not have security context set. It maybe possible to set this to run it more securely.</SecurityNudge>;
        }
    };

    const renderCostOptimisations = () => {
        const recommendations: string[] = [];
        if (!execSpec(workflow).activeDeadlineSeconds) {
            recommendations.push('activeDeadlineSeconds');
        }
        if (!execSpec(workflow).ttlStrategy) {
            recommendations.push('ttlStrategy');
        }
        if (!execSpec(workflow).podGC) {
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
    };

    const renderSummaryTab = () => {
        return (
            <>
                {!workflow ? (
                    <Loading />
                ) : (
                    <div className='workflow-details__container'>
                        <div className='argo-container'>
                            <div className='workflow-details__content'>
                                <WorkflowSummaryPanel workflow={workflow} />
                                {renderSecurityNudge()}
                                {renderCostOptimisations()}
                                {workflow.spec.arguments && workflow.spec.arguments.parameters && (
                                    <React.Fragment>
                                        <h6>Parameters</h6>
                                        <WorkflowParametersPanel parameters={workflow.spec.arguments.parameters} />
                                    </React.Fragment>
                                )}
                                <h5>Artifacts</h5>
                                <WorkflowArtifacts workflow={workflow} archived={false} />
                                <WorkflowResourcePanel workflow={workflow} />
                            </div>
                        </div>
                    </div>
                )}
            </>
        );
    };

    useEffect(() => {
        const retryWatch = new RetryWatch<Workflow>(
            () => services.workflows.watch({name, namespace}),
            () => setError(null),
            e => {
                if (e.type === 'DELETED') {
                    setError(new Error('Workflow deleted'));
                } else {
                    if (hasArtifactGCError(e.object.status.conditions)) {
                        setError(new Error('Artifact garbage collection failed'));
                    }
                    setWorkflow(e.object);
                }
            },
            err => {
                services.workflows
                    .get(namespace, name)
                    .then()
                    .catch(e => {
                        if (e.status === 404) {
                            navigation.goto(historyUrl('archived-workflows', {namespace, name, deep: true}));
                        }
                    });

                setError(err);
            }
        );
        retryWatch.start();
        return () => retryWatch.stop();
    }, [namespace, name]);

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

    const setParameter = (key: string, value: string) => {
        setParameters(previous => {
            return previous?.map(parameter => {
                if (parameter.name === key) {
                    parameter.value = value;
                }
                return parameter;
            });
        });
    };

    const renderSuspendNodeOptions = () => {
        return <SuspendInputs parameters={parameters} nodeId={nodeId} setParameter={setParameter} />;
    };

    const getParametersAsJsonString = () => {
        const outputVariables: {[x: string]: string} = {};
        parameters.forEach(param => {
            outputVariables[param.name] = param.value;
        });
        return JSON.stringify(outputVariables);
    };

    const updateOutputParametersForNodeIfRequired = () => {
        // No need to set outputs on node if there are no parameters
        if (parameters.length > 0) {
            return services.workflows.set(workflow.metadata.name, workflow.metadata.namespace, 'id=' + nodeId, getParametersAsJsonString());
        }
        return Promise.resolve(null);
    };

    const resumeNode = () => {
        return services.workflows.resume(workflow.metadata.name, workflow.metadata.namespace, 'id=' + nodeId);
    };

    const renderResumePopup = () => {
        return popup.confirm('Confirm', renderSuspendNodeOptions).then(yes => {
            if (yes) {
                updateOutputParametersForNodeIfRequired()
                    .then(resumeNode)
                    .catch(setError);
            }
        });
    };

    const ensurePodName = (wf: Workflow, node: NodeStatus, nodeID: string): string => {
        if (workflow && node) {
            let annotations: {[name: string]: string} = {};
            if (typeof workflow.metadata.annotations !== 'undefined') {
                annotations = workflow.metadata.annotations;
            }
            const version = annotations[ANNOTATION_KEY_POD_NAME_VERSION];
            const templateName = getTemplateNameFromNode(node);
            return getPodName(wf.metadata.name, node.name, templateName, node.id, version);
        }

        return nodeID;
    };

    const selectedNode = workflow && workflow.status && workflow.status.nodes && workflow.status.nodes[nodeId];
    const podName = ensurePodName(workflow, selectedNode, nodeId);

    const selectedArtifact = workflow && workflow.status && findArtifact(workflow.status, nodeId);

    return (
        <Page
            title={'Workflow Details'}
            toolbar={{
                breadcrumbs: [
                    {title: 'Workflows', path: uiUrl('workflows')},
                    {title: namespace, path: uiUrl('workflows/' + namespace)},
                    {title: name, path: uiUrl('workflows/' + namespace + '/' + name)}
                ],
                actionMenu: {
                    items: getItems()
                },
                tools: (
                    <div className='workflow-details__topbar-buttons'>
                        <a className={classNames({active: tab === 'summary'})} onClick={() => setTab('summary')}>
                            <i className='fa fa-columns' />
                            {workflow && workflow.status.conditions && hasWarningConditionBadge(workflow.status.conditions) && <span className='badge' />}
                        </a>
                        <a className={classNames({active: tab === 'events'})} onClick={() => setTab('events')}>
                            <i className='fa argo-icon-notification' />
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
            <div className={classNames('workflow-details', {'workflow-details--step-node-expanded': selectedNode, 'workflow-details--artifact-expanded': selectedArtifact})}>
                <ErrorNotice error={error} />
                {(tab === 'summary' && renderSummaryTab()) ||
                    (workflow && (
                        <div>
                            <div className='workflow-details__graph-container'>
                                {(tab === 'workflow' && (
                                    <WorkflowPanel workflowMetadata={workflow.metadata} workflowStatus={workflow.status} selectedNodeId={nodeId} nodeClicked={setNodeId} />
                                )) ||
                                    (tab === 'events' && <EventsPanel namespace={workflow.metadata.namespace} kind='Workflow' name={workflow.metadata.name} />) || (
                                        <WorkflowTimeline workflow={workflow} selectedNodeId={nodeId} nodeClicked={node => setNodeId(node.id)} />
                                    )}
                            </div>
                            <div className='workflow-details__step-info'>
                                <button className='workflow-details__step-info-close' onClick={() => setNodeId(null)}>
                                    <i className='argo-icon-close' />
                                </button>
                                {selectedNode && (
                                    <WorkflowNodeInfo
                                        node={selectedNode}
                                        onTabSelected={setNodePanelView}
                                        selectedTabKey={nodePanelView}
                                        workflow={workflow}
                                        links={links}
                                        onShowContainerLogs={(x, container) => setSidePanel(`logs:${x}:${container}`)}
                                        onShowEvents={() => setSidePanel(`events:${nodeId}`)}
                                        onShowYaml={() => setSidePanel(`yaml:${nodeId}`)}
                                        archived={false}
                                        onResume={() => renderResumePopup()}
                                    />
                                )}
                                {selectedArtifact && (
                                    <ArtifactPanel workflow={workflow} artifact={selectedArtifact} artifactRepository={workflow.status.artifactRepositoryRef.artifactRepository} />
                                )}
                            </div>
                        </div>
                    ))}
            </div>
            {workflow && (
                <SlidingPanel isShown={!!sidePanel} onClose={() => setSidePanel(null)}>
                    {parsedSidePanel.type === 'logs' && (
                        <WorkflowLogsViewer workflow={workflow} initialPodName={podName} nodeId={parsedSidePanel.nodeId} container={parsedSidePanel.container} archived={false} />
                    )}
                    {parsedSidePanel.type === 'events' && <EventsPanel namespace={namespace} kind='Pod' name={parsedSidePanel.nodeId} />}
                    {parsedSidePanel.type === 'share' && <WidgetGallery namespace={namespace} name={name} />}
                    {parsedSidePanel.type === 'yaml' && <WorkflowYamlViewer workflow={workflow} selectedNode={selectedNode} />}
                    {!parsedSidePanel}
                </SlidingPanel>
            )}
        </Page>
    );
};

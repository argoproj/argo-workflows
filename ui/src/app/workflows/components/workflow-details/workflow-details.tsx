import {Page, SlidingPanel} from 'argo-ui';
import * as classNames from 'classnames';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {execSpec, Link, Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {CostOptimisationNudge} from '../../../shared/components/cost-optimisation-nudge';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {SecurityNudge} from '../../../shared/components/security-nudge';
import {hasWarningConditionBadge} from '../../../shared/conditions-panel';
import {Context} from '../../../shared/context';
import {historyUrl} from '../../../shared/history';
import {RetryWatch} from '../../../shared/retry-watch';
import {services} from '../../../shared/services';
import * as Operations from '../../../shared/workflow-operations-map';
import {WorkflowOperations} from '../../../shared/workflow-operations-map';
import {EventsPanel} from '../events-panel';
import {WorkflowArtifacts} from '../workflow-artifacts';
import {WorkflowLogsViewer} from '../workflow-logs-viewer/workflow-logs-viewer';
import {WorkflowNodeInfo} from '../workflow-node-info/workflow-node-info';
import {WorkflowPanel} from '../workflow-panel/workflow-panel';
import {WorkflowParametersPanel} from '../workflow-parameters-panel';
import {WorkflowSummaryPanel} from '../workflow-summary-panel';
import {WorkflowTimeline} from '../workflow-timeline/workflow-timeline';
import {WorkflowYamlViewer} from '../workflow-yaml-viewer/workflow-yaml-viewer';
import {WorkflowResourcePanel} from './workflow-resource-panel';

require('./workflow-details.scss');

function parseSidePanelParam(param: string) {
    const [type, nodeId, container] = (param || '').split(':');
    if (type === 'logs' || type === 'yaml') {
        return {type, nodeId, container: container || 'main'};
    }
    return null;
}

export const WorkflowDetails = ({history, location, match}: RouteComponentProps<any>) => {
    // boiler-plate
    const {navigation, popup} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    const [namespace] = useState(match.params.namespace);
    const [name] = useState(match.params.name);
    const [tab, setTab] = useState(queryParams.get('tab') || 'workflow');
    const [nodeId, setNodeId] = useState(queryParams.get('nodeId'));
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel'));

    useEffect(() => {
        history.push(historyUrl('workflows/{namespace}/{name}', {namespace, name, tab, nodeId, sidePanel}));
    }, [namespace, name, tab, nodeId, sidePanel]);

    const [workflow, setWorkflow] = useState<Workflow>();
    const [links, setLinks] = useState<Link[]>();
    const [error, setError] = useState<Error>();

    useEffect(() => {
        services.info
            .getInfo()
            .then(info => setLinks(info.links))
            .catch(setError);
    }, []);

    const parsedSidePanel = parseSidePanelParam(sidePanel);

    const getItems = () => {
        const workflowOperationsMap: WorkflowOperations = Operations.WorkflowOperationsMap;
        const items = Object.keys(workflowOperationsMap).map(actionName => {
            const workflowOperation = workflowOperationsMap[actionName];
            return {
                title: workflowOperation.title.charAt(0).toUpperCase() + workflowOperation.title.slice(1),
                iconClassName: workflowOperation.iconClassName,
                disabled: workflowOperation.disabled(workflow),
                action: () => {
                    popup
                        .confirm('Confirm', `Are you sure you want to ${workflowOperation.title.toLowerCase()} this workflow?`)
                        .then(() => workflowOperation.action(workflow))
                        .then((wf: Workflow) => {
                            if (workflowOperation.title === 'DELETE') {
                                navigation.goto(uiUrl(`workflows/${workflow.metadata.namespace}`));
                            } else {
                                // TODO - should fix this
                                document.location.href = uiUrl(`workflows/${wf.metadata.namespace}/${wf.metadata.name}`);
                            }
                        })
                        .catch(setError);
                }
            };
        });

        items.push({
            action: () => setSidePanel('logs'),
            disabled: false,
            iconClassName: 'fa fa-file',
            title: 'Logs'
        });

        if (links) {
            links
                .filter(link => link.scope === 'workflow')
                .forEach(link => {
                    items.push({
                        title: link.name,
                        iconClassName: 'fa fa-link',
                        disabled: false,
                        action: () => openLink(link)
                    });
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
                            <h6>Artifacts</h6>
                            <WorkflowArtifacts workflow={workflow} archived={false} />
                            <WorkflowResourcePanel workflow={workflow} />
                        </div>
                    </div>
                )}
            </>
        );
    };

    useEffect(() => {
        services.workflows
            .get(namespace, name)
            .then(setWorkflow)
            .catch(setError);
    }, [namespace, name]);

    useEffect(() => {
        if (!workflow) {
            return;
        }
        const retryWatch = new RetryWatch<Workflow>(
            resourceVersion =>
                services.workflows.watch({
                    name: workflow.metadata.name,
                    namespace: workflow.metadata.namespace,
                    resourceVersion
                }),
            () => setError(null),
            e => setWorkflow(e.object),
            setError
        );
        retryWatch.start(workflow.metadata.resourceVersion);
        return () => retryWatch.stop();
    }, [workflow]);

    const openLink = (link: Link) => {
        const url = link.url
            .replace(/\${metadata\.namespace}/g, workflow.metadata.namespace)
            .replace(/\${metadata\.name}/g, workflow.metadata.name)
            .replace(/\${status\.startedAt}/g, workflow.status.startedAt)
            .replace(/\${status\.finishedAt}/g, workflow.status.finishedAt);
        if ((window.event as MouseEvent).ctrlKey) {
            window.open(url, '_blank');
        } else {
            document.location.href = url;
        }
    };

    const selectedNode = workflow && workflow.status && workflow.status.nodes && workflow.status.nodes[nodeId];
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
            <div className={classNames('workflow-details', {'workflow-details--step-node-expanded': !!selectedNode})}>
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
                                        workflow={workflow}
                                        links={links}
                                        onShowContainerLogs={(_, container) => setSidePanel(`logs:${nodeId}:${container}`)}
                                        onShowYaml={() => setSidePanel(`yaml:${nodeId}`)}
                                        archived={false}
                                    />
                                )}
                            </div>
                        </div>
                    ))}
            </div>
            {workflow && (
                <SlidingPanel isShown={!!sidePanel} onClose={() => setSidePanel(null)}>
                    {sidePanel && parsedSidePanel.type === 'logs' && (
                        <WorkflowLogsViewer workflow={workflow} nodeId={parsedSidePanel.nodeId} container={parsedSidePanel.container} archived={false} />
                    )}
                    {sidePanel && parsedSidePanel.type === 'yaml' && <WorkflowYamlViewer workflow={workflow} selectedNode={selectedNode} />}
                </SlidingPanel>
            )}
        </Page>
    );
};

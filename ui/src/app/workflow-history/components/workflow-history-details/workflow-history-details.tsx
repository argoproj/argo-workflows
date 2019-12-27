import {DataLoader, Page, SlidingPanel} from 'argo-ui';
import * as classNames from 'classnames';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {services} from '../../../shared/services';
import {WorkflowDag, WorkflowTimeline, WorkflowYamlViewer} from '../../../workflows/components';
import {WorkflowArtifacts} from '../../../workflows/components/workflow-artifacts';
import {WorkflowNodeInfo} from '../../../workflows/components/workflow-node-info/workflow-node-info';
import {WorkflowParametersPanel} from '../../../workflows/components/workflow-parameters-panel';
import {WorkflowSummaryPanel} from '../../../workflows/components/workflow-summary-panel';

require('../../../workflows/components/workflow-details/workflow-details.scss');

export class WorkflowHistoryDetails extends React.Component<RouteComponentProps<any>, any> {
    private get namespace() {
        return this.props.match.params.namespace;
    }

    private get uid() {
        return this.props.match.params.uid;
    }

    private get tab() {
        return this.getParam('tab') || 'workflow';
    }

    private set tab(tab) {
        this.setParam('tab', tab);
    }

    private get nodeId() {
        return this.getParam('nodeId');
    }

    private set nodeId(nodeId) {
        this.setParam('nodeId', nodeId);
    }

    private get sidePanel() {
        return this.getParam('sidePanel');
    }

    private set sidePanel(sidePanel) {
        this.setParam('sidePanel', sidePanel);
    }

    public render() {
        return (
            <Page
                title='Workflow History Details'
                toolbar={{
                    actionMenu: {
                        items: [
                            {
                                title: 'Resubmit',
                                iconClassName: 'fa fa-redo',
                                action: () => this.resubmitWorkflowHistory()
                            },
                            {
                                title: 'Delete',
                                iconClassName: 'fa fa-trash',
                                action: () => this.deleteWorkflowHistory()
                            }
                        ]
                    },
                    breadcrumbs: [
                        {
                            title: 'Workflow History',
                            path: uiUrl('workflow-history')
                        },
                        {title: this.namespace + '/' + this.uid}
                    ],
                    tools: (
                        <div className='workflow-details__topbar-buttons'>
                            <a className={classNames({actve: this.tab === 'summary'})} onClick={() => (this.tab = 'summary')}>
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
                <DataLoader load={() => services.workflowHistory.get(this.namespace, this.uid)}>
                    {wf => (
                        <React.Fragment>
                            <div className={classNames('workflow-details', {'workflow-details--step-node-expanded': !!this.nodeId})}>
                                {this.tab === 'summary' ? (
                                    <div className='argo-container'>
                                        <div className='workflow-details__content'>
                                            <WorkflowSummaryPanel workflow={wf} />
                                            {wf.spec.arguments && wf.spec.arguments.parameters && (
                                                <React.Fragment>
                                                    <h6>Parameters</h6>
                                                    <WorkflowParametersPanel parameters={wf.spec.arguments.parameters} />
                                                </React.Fragment>
                                            )}
                                            <h6>Artifacts</h6>
                                            <WorkflowArtifacts workflow={wf} />
                                        </div>
                                    </div>
                                ) : (
                                    <div>
                                        <div className='workflow-details__graph-container'>
                                            {this.tab === 'workflow' ? (
                                                <WorkflowDag workflow={wf} selectedNodeId={this.nodeId} nodeClicked={node => (this.nodeId = node.id)} />
                                            ) : (
                                                <WorkflowTimeline workflow={wf} selectedNodeId={this.nodeId} nodeClicked={node => (this.nodeId = node.id)} />
                                            )}
                                        </div>
                                        {this.nodeId && (
                                            <div className='workflow-details__step-info'>
                                                <button className='workflow-details__step-info-close' onClick={() => (this.nodeId = null)}>
                                                    <i className='argo-icon-close' />
                                                </button>
                                                <WorkflowNodeInfo node={this.node(wf)} workflow={wf} onShowYaml={() => (this.sidePanel = 'yaml')} />
                                            </div>
                                        )}
                                    </div>
                                )}
                                <SlidingPanel isShown={!!this.sidePanel} onClose={() => (this.sidePanel = null)}>
                                    <WorkflowYamlViewer workflow={wf} selectedNode={this.node(wf)} />
                                </SlidingPanel>
                            </div>
                        </React.Fragment>
                    )}
                </DataLoader>
            </Page>
        );
    }

    private getParam(key: string) {
        return new URLSearchParams(this.props.location.search).get(key);
    }

    private setParam(key: string, val: string) {
        let search = document.location.search.split('&').filter(v => !v.startsWith(key + '='));
        if (val !== null) {
            search = search.concat([key + '=' + val]);
        }
        document.location.search = search.join('&');
    }

    private node(wf: models.Workflow) {
        return this.nodeId && wf.status.nodes[this.nodeId];
    }

    private resubmitWorkflowHistory() {
        if (!confirm('Are you sure you want to re-submit this workflow history?')) {
            return;
        }
        services.workflowHistory
            .resubmit(this.namespace, this.uid)
            .catch(e => {
                alert('Failed to resubmit workflow history ' + e);
            })
            .then((wf: models.Workflow) => {
                document.location.href = `/workflows/${wf.metadata.namespace}/${wf.metadata.name}`;
            });
    }

    private deleteWorkflowHistory() {
        if (!confirm('Are you sure you want to delete this workflow history? There is no undo.')) {
            return;
        }
        services.workflowHistory
            .delete(this.namespace, this.uid)
            .catch(e => {
                alert('Failed to delete workflow history ' + e);
            })
            .then(() => {
                document.location.href = '/workflow-history';
            });
    }
}

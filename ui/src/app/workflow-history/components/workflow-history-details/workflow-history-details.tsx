import {DataLoader, Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {services} from '../../../shared/services';
import {WorkflowDag} from '../../../workflows/components';

require('../../../workflows/components/workflow-details/workflow-details.scss');

export class WorkflowHistoryDetails extends React.Component<RouteComponentProps<any>, any> {
    private get namespace() {
        return this.props.match.params.namespace;
    }

    private get uid() {
        return this.props.match.params.uid;
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
                    ]
                }}>
                <div className='workflow-details'>
                    <DataLoader load={() => services.workflowHistory.get(this.namespace, this.uid)}>
                        {workflow => (
                            <div className='workflow-details__graph-container'>
                                <WorkflowDag workflow={workflow} />
                            </div>
                        )}
                    </DataLoader>
                </div>
            </Page>
        );
    }

    private resubmitWorkflowHistory() {
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

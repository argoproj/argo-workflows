import {DataLoader, Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {uiUrl} from '../../../shared/base';
import {services} from '../../../shared/services';
import {WorkflowDag} from '../../../workflows/components';

require('../../../workflows/components/workflow-details/workflow-details.scss');

export class WorkflowHistoryDetails extends React.Component<RouteComponentProps<any>, any> {
    public render() {
        return (
            <Page
                title='Workflow History Details'
                toolbar={{
                    breadcrumbs: [{title: 'Workflow History', path: uiUrl('workflow-history')}, {title: this.props.match.params.namespace + '/' + this.props.match.params.uid}]
                }}>
                <div className='workflow-details'>
                    <DataLoader load={() => services.workflowHistory.get(this.props.match.params.namespace, this.props.match.params.uid)}>
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
}

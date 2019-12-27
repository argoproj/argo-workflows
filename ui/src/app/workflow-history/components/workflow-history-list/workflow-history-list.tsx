import {Page} from 'argo-ui';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {RouteComponentProps} from 'react-router-dom';
import {uiUrl} from '../../../shared/base';

export class WorkflowHistoryList extends React.Component<RouteComponentProps<any>> {
    public static contextTypes = {
        router: PropTypes.object,
        apis: PropTypes.object
    };

    public render() {
        return (
            <Page
                title='Workflow History'
                toolbar={{
                    breadcrumbs: [{title: 'Workflow History', path: uiUrl('workflow-history')}]
                }}>
                <div className='workflow-history-list'>TODO</div>
            </Page>
        );
    }
}

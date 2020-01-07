import {DataLoader, MockupList, Page} from 'argo-ui';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {uiUrl} from '../../../shared/base';
import {services} from '../../../shared/services';
import {WorkflowListItem} from '../../../workflows/components';

export class ArchivedWorkflowList extends React.Component<RouteComponentProps<any>> {
    public static contextTypes = {
        router: PropTypes.object,
        apis: PropTypes.object
    };

    public render() {
        return (
            <Page
                title='Archived Workflows'
                toolbar={{
                    breadcrumbs: [{title: 'Archived Workflows', path: uiUrl('archived-workflow')}]
                }}>
                <DataLoader load={() => services.archivedWorkflows.list()} loadingRenderer={() => <MockupList height={150} marginTop={30} />}>
                    {workflows => (
                        <div className='row'>
                            <div className='columns small-12 xxlarge-2'>
                                {workflows.length === 0 && (
                                    <div className='white-box'>
                                        <h4>No archived workflows</h4>
                                        <p>To record entries you must enabled archiving in configuration.</p>
                                    </div>
                                )}
                                <p>
                                    <i className='fa fa-info-circle' /> Records are created in the archive when a workflow completes.
                                </p>
                                {workflows.map(workflow => (
                                    <div key={workflow.metadata.uid}>
                                        <Link to={uiUrl(`archived-workflows/${workflow.metadata.namespace}/${workflow.metadata.uid}`)}>
                                            <WorkflowListItem workflow={workflow} archived={true} />
                                        </Link>
                                    </div>
                                ))}
                            </div>
                        </div>
                    )}
                </DataLoader>
            </Page>
        );
    }
}

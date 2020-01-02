import {DataLoader, MockupList, Page} from 'argo-ui';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {services} from '../../../shared/services';

export class WorkflowTemplateList extends React.Component<RouteComponentProps<any>> {
    public static contextTypes = {
        router: PropTypes.object,
        apis: PropTypes.object
    };

    public render() {
        return (
            <Page
                title='Templates'
                toolbar={{
                    breadcrumbs: [{title: 'Templates', path: uiUrl('templates')}]
                }}>
                <div className='workflow-template-list'>
                    {/*TODO(simon): remove hardwired 'argo' namespace*/}
                    <DataLoader load={() => services.workflowTemplate.list('argo')} loadingRenderer={() => <MockupList height={150} marginTop={30} />}>
                        {(workflowTemplates: models.WorkflowTemplate[]) => (
                            <div className='row'>
                                <div className='columns small-12 xxlarge-2'>
                                    {workflowTemplates.length === 0 && (
                                        <div className='white-box'>
                                            <h4>No workflow templates</h4>
                                        </div>
                                    )}
                                    {workflowTemplates.map(workflowTemplate => (
                                        <div key={workflowTemplate.metadata.name}>
                                            <Link to={uiUrl(`templates/${workflowTemplate.metadata.namespace}/${workflowTemplate.metadata.name}`)}>
                                                {workflowTemplate.metadata.name}
                                            </Link>
                                        </div>
                                    ))}
                                </div>
                            </div>
                        )}
                    </DataLoader>
                </div>
            </Page>
        );
    }
}

import {DataLoader, NotificationType, Page} from 'argo-ui';
import {AppContext} from 'argo-ui/src/index';
import * as PropTypes from 'prop-types';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {uiUrl} from '../../../shared/base';
import {services} from '../../../shared/services';
import {WorkflowTemplateSummaryPanel} from '../workflow-template-summary-panel';

require('../../../workflows/components/workflow-details/workflow-details.scss');

export class WorkflowTemplateDetails extends React.Component<RouteComponentProps<any>, any> {
    public static contextTypes = {
        router: PropTypes.object,
        apis: PropTypes.object
    };

    private get namespace() {
        return this.props.match.params.namespace;
    }

    private get name() {
        return this.props.match.params.name;
    }

    public render() {
        return (
            <Page
                title='Workflow Template Details'
                toolbar={{
                    actionMenu: {
                        items: [
                            {
                                title: 'Delete',
                                iconClassName: 'fa fa-trash',
                                action: () => this.deleteWorkflowTemplate()
                            }
                        ]
                    },
                    breadcrumbs: [
                        {
                            title: 'Workflow Template',
                            path: uiUrl('templates')
                        },
                        {title: this.namespace + '/' + this.name}
                    ]
                }}>
                <DataLoader load={() => services.workflowTemplate.get(this.name, this.namespace)}>
                    {wfTmpl => (
                        <div className='argo-container'>
                            <div className='workflow-details__content'>
                                <WorkflowTemplateSummaryPanel workflowTemplate={wfTmpl} />
                            </div>
                        </div>
                    )}
                </DataLoader>
            </Page>
        );
    }

    private deleteWorkflowTemplate() {
        if (!confirm('Are you sure you want to delete this workflow template?\nThere is no undo.')) {
            return;
        }
        services.workflowTemplate
            .delete(this.name, this.namespace)
            .catch(e => {
                this.appContext.apis.notifications.show({
                    content: 'Failed to delete workflow template ' + e,
                    type: NotificationType.Error
                });
            })
            .then(() => {
                document.location.href = '/templates';
            });
    }

    private get appContext(): AppContext {
        return this.context as AppContext;
    }
}

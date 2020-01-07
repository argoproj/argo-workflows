import {NotificationType, Page} from 'argo-ui';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {Workflow, WorkflowTemplate} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {services} from '../../../shared/services';
import {WorkflowTemplateSummaryPanel} from '../workflow-template-summary-panel';

require('../../../workflows/components/workflow-details/workflow-details.scss');

interface State {
    template?: WorkflowTemplate;
    error?: Error;
}

export class WorkflowTemplateDetails extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.props.match.params.namespace;
    }

    private get name() {
        return this.props.match.params.name;
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {};
    }

    public componentDidMount(): void {
        services.workflowTemplate
            .get(this.name, this.namespace)
            .then(template => this.setState({template}))
            .catch(error => this.setState({error}));
    }

    public render() {
        if (this.state.error !== undefined) {
            throw this.state.error;
        }
        return (
            <Page
                title='Workflow Template Details'
                toolbar={{
                    actionMenu: {
                        items: [
                            {
                                title: 'Submit',
                                iconClassName: 'fa fa-plus',
                                action: () => this.submitWorkflowTemplate()
                            },
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
                            path: uiUrl('/workflow-templates')
                        },
                        {title: this.namespace + '/' + this.name}
                    ]
                }}>
                <div className='argo-container'>
                    <div className='workflow-details__content'>{this.state.template && <WorkflowTemplateSummaryPanel workflowTemplate={this.state.template} />}</div>
                </div>
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
                document.location.href = '/workflow-templates';
            });
    }

    private submitWorkflowTemplate() {
        const entrypoint = this.state.template.spec.templates[0].name;
        if (!confirm(`Are you sure you want to submit this workflow template?\nEntry-point "${entrypoint}"`)) {
            return;
        }
        services.workflows
            .create(
                {
                    metadata: {
                        generateName: this.state.template.metadata.name,
                        namespace: this.state.template.metadata.namespace
                    },
                    spec: {
                        entrypoint,
                        templates: this.state.template.spec.templates
                    }
                },
                this.namespace
            )
            .catch(e => {
                this.appContext.apis.notifications.show({
                    content: 'Failed to submit template ' + e,
                    type: NotificationType.Error
                });
            })
            .then((workflow: Workflow) => {
                document.location.href = `/workflows/${workflow.metadata.namespace}/${workflow.metadata.name}`;
            });
    }
}

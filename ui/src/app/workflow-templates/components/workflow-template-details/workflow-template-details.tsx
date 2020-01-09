import {NotificationType, Page} from 'argo-ui';
import {SlidingPanel} from 'argo-ui/src/index';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {Workflow, WorkflowTemplate} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {services} from '../../../shared/services';
import WorkflowSubmit from '../../../workflows/components/workflow-submit/workflow-submit';
import {WorkflowTemplateSummaryPanel} from '../workflow-template-summary-panel';

require('../../../workflows/components/workflow-details/workflow-details.scss');

interface State {
    template?: WorkflowTemplate;
    workflow?: Workflow;
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
                                action: () => this.openSubmissionPanel()
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
                            path: uiUrl('workflow-templates')
                        },
                        {title: this.namespace + '/' + this.name}
                    ]
                }}>
                <div className='argo-container'>
                    <div className='workflow-details__content'>{this.renderWorkflowTemplate()}</div>
                </div>
                <SlidingPanel isShown={!!this.state.workflow} onClose={() => this.setState({workflow: null})}>
                    <WorkflowSubmit
                        placeholder={this.state.workflow}
                        onSaved={workflow => (document.location.href = `/workflows/${workflow.metadata.namespace}/${workflow.metadata.name}`)}
                        onError={error => this.setState({error})}
                    />
                </SlidingPanel>
            </Page>
        );
    }

    private renderWorkflowTemplate() {
        if (!this.state.template) {
            return <Loading />;
        }
        return <WorkflowTemplateSummaryPanel workflowTemplate={this.state.template} />;
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

    private openSubmissionPanel() {
        this.setState({
            workflow: {
                metadata: {
                    generateName: this.state.template.metadata.name,
                    namespace: this.state.template.metadata.namespace
                },
                spec: {
                    entrypoint: this.state.template.spec.templates[0].name,
                    templates: this.state.template.spec.templates
                }
            }
        });
    }
}

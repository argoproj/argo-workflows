import {NotificationType, Page} from 'argo-ui';
import {SlidingPanel} from 'argo-ui/src/index';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {Consumer} from '../../../shared/context';
import {services} from '../../../shared/services';
import {SubmitWorkflowPanel} from '../../../workflows/components/submit-workflow-panel';
import {WorkflowTemplateSummaryPanel} from '../workflow-template-summary-panel';

require('../../../workflows/components/workflow-details/workflow-details.scss');

interface State {
    template?: models.WorkflowTemplate;
    error?: Error;
}

export class WorkflowTemplateDetails extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.props.match.params.namespace || '';
    }

    private get name() {
        return this.props.match.params.name;
    }

    private get sidePanel() {
        return this.queryParam('sidePanel');
    }

    private set sidePanel(sidePanel) {
        this.setQueryParams({sidePanel});
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {};
    }

    public componentDidMount(): void {
        services.workflowTemplate
            .get(this.name, this.namespace)
            .then(template => this.setState({error: null, template}))
            .catch(error => this.setState({error}));
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Workflow Template Details'
                        toolbar={{
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Submit',
                                        iconClassName: 'fa fa-plus',
                                        action: () => (this.sidePanel = 'new')
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
                        {this.state.template && (
                            <SlidingPanel isShown={this.sidePanel !== null} onClose={() => (this.sidePanel = null)}>
                                <SubmitWorkflowPanel
                                    kind='WorkflowTemplate'
                                    namespace={this.state.template.metadata.namespace}
                                    name={this.state.template.metadata.name}
                                    entrypoint={this.state.template.spec.entrypoint}
                                    entrypoints={(this.state.template.spec.templates || []).map(t => t.name)}
                                    parameters={this.state.template.spec.arguments.parameters || []}
                                />
                            </SlidingPanel>
                        )}
                    </Page>
                )}
            </Consumer>
        );
    }

    private renderWorkflowTemplate() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} />;
        }
        if (!this.state.template) {
            return <Loading />;
        }
        return <WorkflowTemplateSummaryPanel template={this.state.template} onChange={template => this.setState({template})} />;
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
                document.location.href = uiUrl('workflow-templates');
            });
    }
}

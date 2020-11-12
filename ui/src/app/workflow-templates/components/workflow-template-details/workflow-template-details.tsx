import {Page} from 'argo-ui';
import {SlidingPanel} from 'argo-ui/src/index';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {Status, StatusNotice} from '../../../shared/components/status-notice';
import {services} from '../../../shared/services';
import {SubmitWorkflowPanel} from '../../../workflows/components/submit-workflow-panel';
import {WorkflowTemplateSummaryPanel} from '../workflow-template-summary-panel';

require('../../../workflows/components/workflow-details/workflow-details.scss');

interface State {
    template?: models.WorkflowTemplate;
    status?: Status;
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
            .then(template => this.setState({status: null, template}))
            .catch(error => this.setState({status: error}));
    }

    public render() {
        return (
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
                                title: 'Save',
                                iconClassName: 'fa fa-save',
                                action: () => this.saveWorkflowTemplate()
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
        );
    }

    public saveWorkflowTemplate() {
        services.workflowTemplate
            .update(this.state.template, this.state.template.metadata.name, this.state.template.metadata.namespace)
            .then(template => this.setState({template}))
            .then(() => this.setState({status: 'Succeeded'}))
            .catch(status => this.setState({status}));
    }

    private renderWorkflowTemplate() {
        return (
            <>
                {this.state.status && <StatusNotice status={this.state.status} />}
                {!this.state.template ? (
                    <Loading />
                ) : (
                    <WorkflowTemplateSummaryPanel template={this.state.template} onChange={template => this.setState({template})} onError={status => this.setState({status})} />
                )}
            </>
        );
    }

    private deleteWorkflowTemplate() {
        if (!confirm('Are you sure you want to delete this workflow template?\nThere is no undo.')) {
            return;
        }
        services.workflowTemplate
            .delete(this.name, this.namespace)
            .catch(status => this.setState({status}))
            .then(() => (document.location.href = uiUrl('workflow-templates')));
    }
}

import {NotificationType, Page} from 'argo-ui';
import {SlidingPanel} from 'argo-ui/src/index';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {ResourceSubmit} from '../../../shared/components/resource-submit';
import {Consumer} from '../../../shared/context';
import {services} from '../../../shared/services';
import {ClusterWorkflowTemplateSummaryPanel} from '../cluster-workflow-template-summary-panel';

require('../../../workflows/components/workflow-details/workflow-details.scss');

interface State {
    namespace?: string;
    template?: models.ClusterWorkflowTemplate;
    error?: Error;
}

export class ClusterWorkflowTemplateDetails extends BasePage<RouteComponentProps<any>, State> {
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
        services.info
            .getInfo()
            .then(info => this.setState({namespace: info.managedNamespace || 'default'}))
            .catch(error => this.setState(error));
        services.clusterWorkflowTemplate
            .get(this.name)
            .then(template => this.setState({template}))
            .catch(error => this.setState({error}));
    }

    public render() {
        if (this.state.error !== undefined) {
            throw this.state.error;
        }
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Cluster Workflow Template Details'
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
                                        action: () => this.deleteClusterWorkflowTemplate()
                                    }
                                ]
                            },
                            breadcrumbs: [
                                {
                                    title: 'Cluster Workflow Template',
                                    path: uiUrl('cluster-workflow-templates')
                                },
                                {title: this.name}
                            ]
                        }}>
                        <div className='argo-container'>
                            <div className='workflow-details__content'>{this.renderClusterWorkflowTemplate()}</div>
                        </div>
                        {this.state.template && (
                            <SlidingPanel isShown={this.sidePanel !== null} onClose={() => (this.sidePanel = null)}>
                                <ResourceSubmit<models.Workflow>
                                    resourceName={'Workflow'}
                                    defaultResource={this.getWorkflow(this.state.template)}
                                    onSubmit={wfValue => {
                                        return services.workflows
                                            .create(wfValue, wfValue.metadata.namespace)
                                            .then(workflow => ctx.navigation.goto(uiUrl(`workflows/${workflow.metadata.namespace}/${workflow.metadata.name}`)));
                                    }}
                                />
                            </SlidingPanel>
                        )}
                    </Page>
                )}
            </Consumer>
        );
    }

    private renderClusterWorkflowTemplate() {
        if (!this.state.template) {
            return <Loading />;
        }
        return <ClusterWorkflowTemplateSummaryPanel template={this.state.template} onChange={template => this.setState({template})} onError={error => this.setState({error})} />;
    }

    private deleteClusterWorkflowTemplate() {
        if (!confirm('Are you sure you want to delete this cluster workflow template?\nThere is no undo.')) {
            return;
        }
        services.clusterWorkflowTemplate
            .delete(this.name)
            .catch(e => {
                this.appContext.apis.notifications.show({
                    content: 'Failed to delete cluster workflow template ' + e,
                    type: NotificationType.Error
                });
            })
            .then(() => {
                document.location.href = uiUrl('cluster-workflow-templates');
            });
    }

    private getWorkflow(template: models.ClusterWorkflowTemplate): models.Workflow {
        return {
            metadata: {
                generateName: template.metadata.name + '-',
                namespace: this.state.namespace
            },
            spec: {
                entrypoint: !!template.spec.templates ? template.spec.templates[0].name : '',
                workflowTemplateRef: {
                    name: template.metadata.name,
                    clusterScope: true
                }
            }
        };
    }
}

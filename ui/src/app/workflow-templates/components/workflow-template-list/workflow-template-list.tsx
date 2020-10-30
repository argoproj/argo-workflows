import {Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {ExampleManifests} from '../../../shared/components/example-manifests';
import {Loading} from '../../../shared/components/loading';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {ResourceEditor} from '../../../shared/components/resource-editor/resource-editor';
import {Timestamp} from '../../../shared/components/timestamp';
import {ZeroState} from '../../../shared/components/zero-state';
import {Consumer} from '../../../shared/context';
import {exampleWorkflowTemplate} from '../../../shared/examples';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';

require('./workflow-template-list.scss');

interface State {
    namespace: string;
    templates?: models.WorkflowTemplate[];
    error?: Error;
}

export class WorkflowTemplateList extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.state.namespace;
    }

    private set namespace(namespace: string) {
        this.fetchWorkflowTemplates(namespace);
    }

    private get sidePanel() {
        return this.queryParam('sidePanel');
    }

    private set sidePanel(sidePanel) {
        this.setQueryParams({sidePanel});
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {namespace: this.props.match.params.namespace || ''};
    }

    public componentDidMount(): void {
        this.fetchWorkflowTemplates(this.namespace);
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Workflow Templates'
                        toolbar={{
                            breadcrumbs: [{title: 'Workflow Templates', path: uiUrl('workflow-templates')}],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Create New Workflow Template',
                                        iconClassName: 'fa fa-plus',
                                        action: () => (this.sidePanel = 'new')
                                    }
                                ]
                            },
                            tools: [<NamespaceFilter key='namespace-filter' value={this.namespace} onChange={namespace => (this.namespace = namespace)} />]
                        }}>
                        {this.renderTemplates()}
                        <SlidingPanel isShown={this.sidePanel !== null} onClose={() => (this.sidePanel = null)}>
                            <ResourceEditor
                                title='New Workflow Template'
                                kind='WorkflowTemplate'
                                upload={true}
                                namespace={this.namespace || 'default'}
                                value={exampleWorkflowTemplate()}
                                onSubmit={wfTmpl =>
                                    services.workflowTemplate
                                        .create(wfTmpl, wfTmpl.metadata.namespace)
                                        .then(wf => ctx.navigation.goto(uiUrl(`workflow-templates/${wf.metadata.namespace}/${wf.metadata.name}`)))
                                }
                                editing={true}
                            />
                            <p>
                                <ExampleManifests />.
                            </p>
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
    }

    private saveHistory() {
        this.url = uiUrl('workflow-templates/' + this.namespace || '');
        Utils.setCurrentNamespace(this.namespace);
    }

    private fetchWorkflowTemplates(namespace: string): void {
        services.workflowTemplate
            .list(namespace)
            .then(templates => this.setState({error: null, namespace, templates}, this.saveHistory))
            .catch(error => this.setState({error}));
    }

    private renderTemplates() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} style={{margin: 20}} />;
        }
        if (!this.state.templates) {
            return <Loading />;
        }
        const learnMore = <a href='https://argoproj.github.io/argo/workflow-templates/'>Learn more</a>;
        if (this.state.templates.length === 0) {
            return (
                <ZeroState title='No workflow templates'>
                    <p>You can create new templates here or using the CLI.</p>
                    <p>
                        <ExampleManifests />. {learnMore}.
                    </p>
                </ZeroState>
            );
        }
        return (
            <div className='row'>
                <div className='columns small-12'>
                    <div className='argo-table-list'>
                        <div className='row argo-table-list__head'>
                            <div className='columns small-1' />
                            <div className='columns small-5'>NAME</div>
                            <div className='columns small-3'>NAMESPACE</div>
                            <div className='columns small-3'>CREATED</div>
                        </div>
                        {this.state.templates.map(t => (
                            <Link
                                className='row argo-table-list__row'
                                key={`${t.metadata.namespace}/${t.metadata.name}`}
                                to={uiUrl(`workflow-templates/${t.metadata.namespace}/${t.metadata.name}`)}>
                                <div className='columns small-1'>
                                    <i className='fa fa-clone' />
                                </div>
                                <div className='columns small-5'>{t.metadata.name}</div>
                                <div className='columns small-3'>{t.metadata.namespace}</div>
                                <div className='columns small-3'>
                                    <Timestamp date={t.metadata.creationTimestamp} />
                                </div>
                            </Link>
                        ))}
                    </div>
                    <p>
                        <i className='fa fa-info-circle' /> Workflow templates are reusable templates you can create new workflows from. <ExampleManifests />. {learnMore}.
                    </p>
                </div>
            </div>
        );
    }
}

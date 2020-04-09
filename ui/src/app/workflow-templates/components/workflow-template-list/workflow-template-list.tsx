import {Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {ResourceSubmit} from '../../../shared/components/resource-submit';
import {Timestamp} from '../../../shared/components/timestamp';
import {ZeroState} from '../../../shared/components/zero-state';
import {Consumer} from '../../../shared/context';
import {exampleWorkflowTemplate} from '../../../shared/examples';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';

require('./workflow-template-list.scss');

interface State {
    loading: boolean;
    namespace: string;
    templates?: models.WorkflowTemplate[];
    error?: Error;
}

export class WorkflowTemplateList extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.state.namespace;
    }

    private set namespace(namespace: string) {
        this.setState({namespace});
        history.pushState(null, '', uiUrl('workflow-templates/' + namespace));
        this.fetchWorkflowTemplates();
        Utils.setCurrentNamespace(namespace);
    }

    private get sidePanel() {
        return this.queryParam('sidePanel');
    }

    private set sidePanel(sidePanel) {
        this.setQueryParams({sidePanel});
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {loading: true, namespace: this.props.match.params.namespace || Utils.getCurrentNamespace() || ''};
    }

    public componentDidMount(): void {
        this.fetchWorkflowTemplates();
    }

    public render() {
        if (this.state.loading) {
            return <Loading />;
        }
        if (this.state.error) {
            throw this.state.error;
        }
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
                            <ResourceSubmit<models.WorkflowTemplate>
                                resourceName={'Workflow Template'}
                                defaultResource={exampleWorkflowTemplate(this.namespace || 'default')}
                                validate={wfValue => {
                                    if (!wfValue || !wfValue.metadata) {
                                        return {valid: false, message: 'Invalid WorkflowTemplate definition'};
                                    }
                                    if (wfValue.metadata.namespace === undefined || wfValue.metadata.namespace === '') {
                                        return {valid: false, message: 'Namespace is missing'};
                                    }
                                    return {valid: true};
                                }}
                                onSubmit={wfTmpl => {
                                    return services.workflowTemplate
                                        .create(wfTmpl, wfTmpl.metadata.namespace)
                                        .then(wf => ctx.navigation.goto(uiUrl(`workflow-templates/${wf.metadata.namespace}/${wf.metadata.name}`)));
                                }}
                            />
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
    }

    private fetchWorkflowTemplates(): void {
        services.info
            .get()
            .then(info => {
                if (info.managedNamespace && info.managedNamespace !== this.namespace) {
                    this.namespace = info.managedNamespace;
                }
                return services.workflowTemplate.list(this.namespace);
            })
            .then(templates => this.setState({templates, loading: false}))
            .catch(error => this.setState({error, loading: false}));
    }

    private renderTemplates() {
        if (!this.state.templates) {
            return <Loading />;
        }
        const learnMore = <a href='https://github.com/argoproj/argo/blob/master/docs/workflow-templates.md'>Learn more</a>;
        if (this.state.templates.length === 0) {
            return (
                <ZeroState title='No workflow templates'>
                    <p>You can create new templates here or using the CLI.</p>
                    <p>{learnMore}.</p>
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
                        <i className='fa fa-info-circle' /> Workflow templates are reusable templates you can create new workflows from. {learnMore}.
                    </p>
                </div>
            </div>
        );
    }
}

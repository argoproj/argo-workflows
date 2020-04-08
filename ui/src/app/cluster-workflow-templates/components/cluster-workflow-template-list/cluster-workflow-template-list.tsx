import {Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {ResourceSubmit} from '../../../shared/components/resource-submit';
import {Timestamp} from '../../../shared/components/timestamp';
import {ZeroState} from '../../../shared/components/zero-state';
import {Consumer} from '../../../shared/context';
import {exampleClusterWorkflowTemplate} from '../../../shared/examples';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';

require('./cluster-workflow-template-list.scss');

interface State {
    loading: boolean;
    namespace: string;
    templates?: models.ClusterWorkflowTemplate[];
    error?: Error;
}

export class ClusterWorkflowTemplateList extends BasePage<RouteComponentProps<any>, State> {
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
        this.fetchClusterWorkflowTemplates();
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
                        title='Cluster Workflow Templates'
                        toolbar={{
                            breadcrumbs: [{title: 'Cluster Workflow Templates', path: uiUrl('cluster-workflow-templates')}],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Create New Cluster Workflow Template',
                                        iconClassName: 'fa fa-plus',
                                        action: () => (this.sidePanel = 'new')
                                    }
                                ]
                            }
                        }}>
                        {this.renderTemplates()}
                        <SlidingPanel isShown={this.sidePanel !== null} onClose={() => (this.sidePanel = null)}>
                            <ResourceSubmit<models.WorkflowTemplate>
                                resourceName={'Cluster Workflow Template'}
                                defaultResource={exampleClusterWorkflowTemplate()}
                                validate={wfTmpl => {
                                    if (!wfTmpl || !wfTmpl.metadata) {
                                        return {valid: false, message: 'Invalid ClusterWorkflowTemplate definition'};
                                    }
                                    return {valid: true};
                                }}
                                onSubmit={wfTmpl => {
                                    return services.clusterWorkflowTemplate.create(wfTmpl).then(wf => ctx.navigation.goto(uiUrl(`cluster-workflow-templates/${wf.metadata.name}`)));
                                }}
                            />
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
    }

    private fetchClusterWorkflowTemplates(): void {
        services.info
            .get()
            .then(info => {
                return services.clusterWorkflowTemplate.list();
            })
            .then(templates => this.setState({templates, loading: false}))
            .catch(error => this.setState({error, loading: false}));
    }

    private renderTemplates() {
        if (!this.state.templates) {
            return <Loading />;
        }
        const learnMore = <a href='https://github.com/argoproj/argo/blob/master/docs/cluster-workflow-templates.md'>Learn more</a>;
        if (this.state.templates.length === 0) {
            return (
                <ZeroState title='No cluster workflow templates'>
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
                            <div className='columns small-3'>CREATED</div>
                        </div>
                        {this.state.templates.map(t => (
                            <Link
                                className='row argo-table-list__row'
                                key={`${t.metadata.namespace}/${t.metadata.name}`}
                                to={uiUrl(`cluster-workflow-templates/${t.metadata.name}`)}>
                                <div className='columns small-1'>
                                    <i className='fa fa-clone' />
                                </div>
                                <div className='columns small-5'>{t.metadata.name}</div>
                                <div className='columns small-3'>
                                    <Timestamp date={t.metadata.creationTimestamp} />
                                </div>
                            </Link>
                        ))}
                    </div>
                    <p>
                        <i className='fa fa-info-circle' /> Cluster scoped Workflow templates are reusable templates you can create new workflows from. {learnMore}.
                    </p>
                </div>
            </div>
        );
    }
}

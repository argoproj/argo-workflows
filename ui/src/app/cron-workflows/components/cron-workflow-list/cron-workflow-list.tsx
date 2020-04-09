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
import {exampleCronWorkflow} from '../../../shared/examples';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';

require('./cron-workflow-list.scss');

interface State {
    loading: boolean;
    namespace: string;
    cronWorkflows?: models.CronWorkflow[];
    error?: Error;
}

export class CronWorkflowList extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.state.namespace;
    }

    private set namespace(namespace: string) {
        this.setState({namespace});
        history.pushState(null, '', uiUrl('cron-workflows/' + namespace));
        this.fetchCronWorkflows();
        Utils.setCurrentNamespace(namespace);
    }

    private get sidePanel() {
        return this.queryParam('sidePanel');
    }

    private set sidePanel(sidePanel) {
        this.setQueryParams({sidePanel});
    }
    constructor(props: any) {
        super(props);
        this.state = {loading: true, namespace: this.props.match.params.namespace || Utils.getCurrentNamespace() || ''};
    }

    public componentDidMount(): void {
        this.fetchCronWorkflows();
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
                        title='Cron Workflows'
                        toolbar={{
                            breadcrumbs: [{title: 'Cron Workflows', path: uiUrl('cron-workflows')}],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Create New Cron Workflow',
                                        iconClassName: 'fa fa-plus',
                                        action: () => (this.sidePanel = 'new')
                                    }
                                ]
                            },
                            tools: [<NamespaceFilter key='namespace-filter' value={this.namespace} onChange={namespace => (this.namespace = namespace)} />]
                        }}>
                        <div className='row'>
                            <div className='columns small-12'>{this.renderCronWorkflows()}</div>
                        </div>
                        <SlidingPanel isShown={this.sidePanel !== null} onClose={() => (this.sidePanel = null)}>
                            <ResourceSubmit<models.CronWorkflow>
                                resourceName={'Cron Workflow'}
                                defaultResource={exampleCronWorkflow(this.namespace)}
                                validate={wfValue => {
                                    if (!wfValue || !wfValue.metadata) {
                                        return {valid: false, message: 'Invalid CronWorkflow definition'};
                                    }
                                    if (wfValue.metadata.namespace === undefined || wfValue.metadata.namespace === '') {
                                        return {valid: false, message: 'Namespace is missing'};
                                    }
                                    return {valid: true};
                                }}
                                onSubmit={cronWf => {
                                    return services.cronWorkflows
                                        .create(cronWf, cronWf.metadata.namespace)
                                        .then(res => ctx.navigation.goto(uiUrl(`cron-workflows/${res.metadata.namespace}/${res.metadata.name}`)));
                                }}
                            />
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
    }

    private fetchCronWorkflows(): void {
        services.info
            .get()
            .then(info => {
                if (info.managedNamespace && info.managedNamespace !== this.namespace) {
                    this.namespace = info.managedNamespace;
                }
                return services.cronWorkflows.list(this.namespace);
            })
            .then(cronWorkflows => this.setState({cronWorkflows, loading: false}))
            .catch(error => this.setState({error, loading: false}));
    }

    private renderCronWorkflows() {
        if (!this.state.cronWorkflows) {
            return <Loading />;
        }
        const learnMore = <a href='https://github.com/argoproj/argo/blob/master/docs/cron-workflows.md'>Learn more</a>;
        if (this.state.cronWorkflows.length === 0) {
            return (
                <ZeroState title='No cron workflows'>
                    <p>You can create new cron workflows here or using the CLI.</p>
                    <p>{learnMore}.</p>
                </ZeroState>
            );
        }
        return (
            <>
                <div className='argo-table-list'>
                    <div className='row argo-table-list__head'>
                        <div className='columns small-1' />
                        <div className='columns small-3'>NAME</div>
                        <div className='columns small-3'>NAMESPACE</div>
                        <div className='columns small-2'>SCHEDULE</div>
                        <div className='columns small-3'>CREATED</div>
                    </div>
                    {this.state.cronWorkflows.map(w => (
                        <Link
                            className='row argo-table-list__row'
                            key={`${w.metadata.namespace}/${w.metadata.name}`}
                            to={uiUrl(`cron-workflows/${w.metadata.namespace}/${w.metadata.name}`)}>
                            <div className='columns small-1'>
                                <i className='fa fa-clock' />
                            </div>
                            <div className='columns small-3'>{w.metadata.name}</div>
                            <div className='columns small-3'>{w.metadata.namespace}</div>
                            <div className='columns small-2'>{w.spec.schedule}</div>
                            <div className='columns small-3'>
                                <Timestamp date={w.metadata.creationTimestamp} />
                            </div>
                        </Link>
                    ))}
                </div>
                <p>
                    <i className='fa fa-info-circle' /> Cron workflows are workflows that run on a preset schedule. {learnMore}.
                </p>
            </>
        );
    }
}

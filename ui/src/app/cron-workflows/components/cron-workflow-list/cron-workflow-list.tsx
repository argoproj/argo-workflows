import {Page, SlidingPanel, Ticker} from 'argo-ui';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {ExampleManifests} from '../../../shared/components/example-manifests';
import {InfoIcon} from '../../../shared/components/fa-icons';
import {Loading} from '../../../shared/components/loading';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {Timestamp} from '../../../shared/components/timestamp';
import {ZeroState} from '../../../shared/components/zero-state';
import {Consumer} from '../../../shared/context';
import {getNextScheduledTime} from '../../../shared/cron';
import {Footnote} from '../../../shared/footnote';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';
import {CronWorkflowCreator} from '../cron-workflow-creator';
import {PrettySchedule} from '../pretty-schedule';

require('./cron-workflow-list.scss');

interface State {
    namespace: string;
    cronWorkflows?: models.CronWorkflow[];
    error?: Error;
}

export class CronWorkflowList extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.state.namespace;
    }

    private set namespace(namespace: string) {
        this.fetchCronWorkflows(namespace);
    }

    private get sidePanel() {
        return this.queryParam('sidePanel');
    }

    private set sidePanel(sidePanel) {
        this.setQueryParams({sidePanel});
    }

    constructor(props: any) {
        super(props);
        this.state = {namespace: Utils.getNamespace(this.props.match.params.namespace) || ''};
    }

    public componentDidMount(): void {
        this.fetchCronWorkflows(this.state.namespace);
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Cron Workflows'
                        toolbar={{
                            breadcrumbs: [
                                {title: 'Cron Workflows', path: uiUrl('cron-workflows')},
                                {title: this.namespace, path: uiUrl('cron-workflows/' + this.namespace)}
                            ],
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
                        {this.renderCronWorkflows()}
                        <SlidingPanel isShown={this.sidePanel !== null} onClose={() => (this.sidePanel = null)}>
                            <CronWorkflowCreator
                                namespace={this.namespace}
                                onCreate={cronWorkflow => ctx.navigation.goto(uiUrl('cron-workflows/' + cronWorkflow.metadata.namespace + '/' + cronWorkflow.metadata.name))}
                            />
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
    }

    private saveHistory() {
        const newNamespace = Utils.managedNamespace ? '' : this.state.namespace;
        this.url = uiUrl('cron-workflows' + (newNamespace ? '/' + newNamespace : ''));
        Utils.currentNamespace = this.state.namespace;
    }

    private fetchCronWorkflows(namespace: string): void {
        services.cronWorkflows
            .list(namespace)
            .then(cronWorkflows => this.setState({error: null, namespace, cronWorkflows}, this.saveHistory))
            .catch(error => this.setState({error}));
    }

    private renderCronWorkflows() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} />;
        }
        if (!this.state.cronWorkflows) {
            return <Loading />;
        }
        const learnMore = <a href='https://argoproj.github.io/argo-workflows/cron-workflows/'>Learn more</a>;
        if (this.state.cronWorkflows.length === 0) {
            return (
                <ZeroState title='No cron workflows'>
                    <p>You can create new cron workflows here or using the CLI.</p>
                    <p>
                        <ExampleManifests />. {learnMore}.
                    </p>
                </ZeroState>
            );
        }
        return (
            <>
                <div className='argo-table-list'>
                    <div className='row argo-table-list__head'>
                        <div className='columns small-1' />
                        <div className='columns small-3'>NAME</div>
                        <div className='columns small-2'>NAMESPACE</div>
                        <div className='columns small-1'>SCHEDULE</div>
                        <div className='columns small-3' />
                        <div className='columns small-1'>CREATED</div>
                        <div className='columns small-1'>NEXT RUN</div>
                    </div>
                    {this.state.cronWorkflows.map(w => (
                        <Link
                            className='row argo-table-list__row'
                            key={`${w.metadata.namespace}/${w.metadata.name}`}
                            to={uiUrl(`cron-workflows/${w.metadata.namespace}/${w.metadata.name}`)}>
                            <div className='columns small-1'>{w.spec.suspend ? <i className='fa fa-pause' /> : <i className='fa fa-clock' />}</div>
                            <div className='columns small-3'>{w.metadata.name}</div>
                            <div className='columns small-2'>{w.metadata.namespace}</div>
                            <div className='columns small-1'>{w.spec.schedule}</div>
                            <div className='columns small-3'>
                                <PrettySchedule schedule={w.spec.schedule} />
                            </div>
                            <div className='columns small-1'>
                                <Timestamp date={w.metadata.creationTimestamp} />
                            </div>
                            <div className='columns small-1'>
                                {w.spec.suspend ? '' : <Ticker intervalMs={1000}>{() => <Timestamp date={getNextScheduledTime(w.spec.schedule, w.spec.timezone)} />}</Ticker>}
                            </div>
                        </Link>
                    ))}
                </div>
                <Footnote>
                    <InfoIcon /> Cron workflows are workflows that run on a preset schedule. Next scheduled run assumes workflow-controller is in UTC. <ExampleManifests />.{' '}
                    {learnMore}.
                </Footnote>
            </>
        );
    }
}

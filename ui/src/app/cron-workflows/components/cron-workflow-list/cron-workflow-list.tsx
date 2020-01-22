import {Page, SlidingPanel} from 'argo-ui';
import * as React from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import * as models from '../../../../models';
import {CronWorkflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {Loading} from '../../../shared/components/loading';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {Timestamp} from '../../../shared/components/timestamp';
import {YamlEditor} from '../../../shared/components/yaml/yaml-editor';
import {ZeroState} from '../../../shared/components/zero-state';
import {Consumer} from '../../../shared/context';
import {exampleCronWorkflow} from '../../../shared/examples';
import {services} from '../../../shared/services';
require('./cron-workflow-list.scss');

interface State {
    cronWorkflows?: models.CronWorkflow[];
    error?: Error;
}

export class CronWorkflowList extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.queryParam('namespace') || '';
    }

    private set namespace(namespace) {
        this.setQueryParams({namespace});
    }

    private get sidePanel() {
        return this.queryParam('sidePanel');
    }

    private set sidePanel(sidePanel) {
        this.setQueryParams({sidePanel});
    }
    constructor(props: any) {
        super(props);
        this.state = {};
    }

    public componentDidMount(): void {
        services.cronWorkflows
            .list(this.namespace)
            .then(cronWorkflows => this.setState({cronWorkflows}))
            .catch(error => this.setState({error}));
    }

    public render() {
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
                            <div className='columns small-12 xxlarge-2'>{this.renderCronWorkflows()}</div>
                        </div>
                        <SlidingPanel isShown={this.sidePanel !== null} onClose={() => (this.sidePanel = null)}>
                            <YamlEditor
                                title='Create New Cron Workflow'
                                editing={true}
                                value={exampleCronWorkflow(this.namespace)}
                                onSubmit={(value: CronWorkflow) => {
                                    return services.cronWorkflows
                                        .create(value, value.metadata.namespace)
                                        .then(res => ctx.navigation.goto(`/cron-workflows/${res.metadata.namespace}/${res.metadata.name}`))
                                        .catch(error => this.setState({error}));
                                }}
                            />
                        </SlidingPanel>
                    </Page>
                )}
            </Consumer>
        );
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

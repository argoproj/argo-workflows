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
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';

require('./event-source-list.scss');

interface State {
    namespace: string;
    eventSources?: models.EventSource[];
    error?: Error;
}

export class EventSourceList extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.state.namespace;
    }

    private set namespace(namespace: string) {
        this.fetchEventSource(namespace);
    }

    private get sidePanel() {
        return this.queryParam('sidePanel');
    }

    private set sidePanel(sidePanel) {
        this.setQueryParams({sidePanel});
    }

    constructor(props: any) {
        super(props);
        this.state = {namespace: this.props.match.params.namespace || ''};
    }

    public componentDidMount(): void {
        this.fetchEventSource(this.state.namespace);
    }

    public render() {
        return (
            <Consumer>
                {ctx => (
                    <Page
                        title='Event Sources'
                        toolbar={{
                            breadcrumbs: [{title: 'Event Sources', path: uiUrl('event-sources')}],
                            actionMenu: {
                                items: [
                                    {
                                        title: 'Create New Event Source',
                                        iconClassName: 'fa fa-plus',
                                        action: () => (this.sidePanel = 'new')
                                    }
                                ]
                            },
                            tools: [<NamespaceFilter key='namespace-filter' value={this.namespace} onChange={namespace => (this.namespace = namespace)} />]
                        }}>
                        <div className='row'>
                            <div className='columns small-12'>{this.renderEventSources()}</div>
                        </div>
                        <SlidingPanel isShown={this.sidePanel !== null} onClose={() => (this.sidePanel = null)}>
                            {/*<ResourceEditor*/}
                            {/*    title={'New Event Source'}*/}
                            {/*    namespace={this.namespace}*/}
                            {/*    // value={exampleCronWorkflow()}*/}
                            {/*    // onSubmit={cronWf =>*/}
                            {/*    //     services.cronWorkflows*/}
                            {/*    //         .create(cronWf, cronWf.metadata.namespace || this.namespace)*/}
                            {/*    //         .then(res => ctx.navigation.goto(uiUrl(`cron-workflows/${res.metadata.namespace}/${res.metadata.name}`)))*/}
                            {/*    // }*/}
                            {/*    upload={true}*/}
                            {/*    editing={true}*/}
                            {/*    kind='CronWorkflow'*/}
                            {/*/>*/}
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
        this.url = uiUrl('event-sources/' + this.state.namespace || '');
        Utils.setCurrentNamespace(this.state.namespace);
    }

    private fetchEventSource(namespace: string): void {
        services.eventSource
            .list(namespace)
            .then( eventSources => this.setState({error: null, namespace, eventSources}, this.saveHistory))
            .catch(error => this.setState({error}));
    }

    private renderEventSources() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} style={{margin: 20}} />;
        }
        if (!this.state.eventSources) {
            return <Loading />;
        }
        // TO-DO bala
        // const learnMore = <a href='https://argoproj.github.io/argo/cron-workflows/'>Learn more</a>;
        if (this.state.eventSources.length === 0) {
            return (
                <ZeroState title='No cron workflows'>
                    <p>You can create new event source here or using the CLI.</p>
                    <p>
                        {/*<ExampleManifests />. {learnMore}.*/}
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
                        <div className='columns small-3'>NAMESPACE</div>
                        <div className='columns small-3'>CREATED</div>
                    </div>
                    {this.state.eventSources.map(w => (
                        <Link
                            className='row argo-table-list__row'
                            key={`${w.metadata.namespace}/${w.metadata.name}`}
                            to={uiUrl(`event-sources/${w.metadata.namespace}/${w.metadata.name}`)}>
                            <div className='columns small-1'> <i className='fas fa-bolt' /> </div>
                            <div className='columns small-3'>{w.metadata.name}</div>
                            <div className='columns small-3'>{w.metadata.namespace}</div>
                            <div className='columns small-3'>
                                <Timestamp date={w.metadata.creationTimestamp} />
                            </div>
                        </Link>
                    ))}
                </div>
            </>
        );
    }
}

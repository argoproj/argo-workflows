import {Page, SlidingPanel, Tabs} from 'argo-ui/src/index';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {Subscription} from 'rxjs';
import {Condition} from '../../../../models';
import {EventSource, eventSourceTypes} from '../../../../models/event-source';
import {Sensor, triggerTypes} from '../../../../models/sensor';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {GraphPanel} from '../../../shared/components/graph/graph-panel';
import {Graph} from '../../../shared/components/graph/types';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {ResourceEditor} from '../../../shared/components/resource-editor/resource-editor';
import {ZeroState} from '../../../shared/components/zero-state';
import {ListWatch} from '../../../shared/list-watch';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';
import {EventsPanel} from '../../../workflows/components/events-panel';
import {FullHeightLogsViewer} from '../../../workflows/components/workflow-logs-viewer/full-height-logs-viewer';
import {icons} from './icons';
import {ID, Type} from './id';

require('../../../workflows/components/workflow-details/workflow-details.scss');

interface State {
    namespace: string;
    markActivations: boolean;
    selectedId?: string;
    tab?: string;
    error?: Error;
    eventSources?: EventSource[];
    sensors?: Sensor[];
    active: {[id: string]: any};
}

const status = (r: {status?: {conditions?: Condition[]}}) => {
    if (!r.status || !r.status.conditions) {
        return '';
    }
    return !!r.status.conditions.find(c => c.status !== 'True') ? 'Pending' : 'Listening';
};

const types = () => {
    const v: {[label: string]: boolean} = {sensor: true, conditions: true};
    Object.keys(eventSourceTypes)
        .concat(Object.keys(triggerTypes))
        .forEach(label => (v[label] = true));
    return v;
};

export class NamespaceDetails extends BasePage<RouteComponentProps<any>, State> {
    private markActivationsSubscriptions: Subscription[];
    private eventSourceListWatch: ListWatch<EventSource>;
    private sensorListWatch: ListWatch<Sensor>;

    private set selectedId(selectedId: string) {
        this.setState({selectedId}, this.saveHistory);
    }

    private get selectedId() {
        return this.state.selectedId;
    }

    private get markActivations() {
        return this.state.markActivations;
    }

    private set markActivations(markActivations: boolean) {
        if (markActivations) {
            this.startMarkingActivations();
        } else {
            this.stopMarkingActivations();
        }
        this.setState({markActivations}, this.saveHistory);
    }

    private get tab() {
        return this.state.tab;
    }

    private set tab(tab: string) {
        this.setState({tab}, this.saveHistory);
    }

    private get namespace() {
        return this.state.namespace;
    }

    private set namespace(namespace: string) {
        this.setState({namespace}, () => {
            this.saveHistory();
            this.fetch();
        });
    }

    private get graph(): Graph {
        const graph = new Graph();
        (this.state.eventSources || []).forEach(eventSource => {
            Object.entries(eventSource.spec)
                .filter(([typeKey]) => ['template', 'service'].indexOf(typeKey) < 0)
                .forEach(([typeKey, type]) => {
                    Object.keys(type).forEach(key => {
                        const eventId = ID.join({
                            type: 'EventSource',
                            namespace: eventSource.metadata.namespace,
                            name: eventSource.metadata.name,
                            key
                        });
                        graph.nodes.set(eventId, {
                            type: typeKey,
                            label: key,
                            classNames: status(eventSource),
                            icon: icons[eventSourceTypes[typeKey] + 'EventSource']
                        });
                    });
                });
        });
        (this.state.sensors || []).forEach(sensor => {
            const sensorId = ID.join({type: 'Sensor', namespace: sensor.metadata.namespace, name: sensor.metadata.name});
            graph.nodes.set(sensorId, {
                type: 'sensor',
                label: sensor.metadata.name,
                icon: icons.Sensor,
                classNames: status(sensor)
            });
            (sensor.spec.dependencies || []).forEach(d => {
                const eventId = ID.join({
                    type: 'EventSource',
                    namespace: sensor.metadata.namespace,
                    name: d.eventSourceName,
                    key: d.eventName
                });
                graph.edges.set({v: eventId, w: sensorId}, {label: d.name, classNames: this.edgeClassNames(eventId)});
            });
            (sensor.spec.triggers || [])
                .map(t => t.template)
                .filter(template => template)
                .forEach(template => {
                    const triggerTypeKey = Object.keys(template).filter(t => ['name', 'conditions'].indexOf(t) === -1)[0];
                    const triggerId = ID.join({
                        type: 'Trigger',
                        namespace: sensor.metadata.namespace,
                        name: sensor.metadata.name,
                        key: template.name
                    });
                    graph.nodes.set(triggerId, {
                        label: template.name,
                        type: triggerTypeKey,
                        classNames: status(sensor),
                        icon: icons[triggerTypes[triggerTypeKey] + 'Trigger']
                    });
                    if (template.conditions) {
                        const conditionsId = ID.join({
                            type: 'Conditions',
                            namespace: sensor.metadata.namespace,
                            name: sensor.metadata.name,
                            key: template.conditions
                        });
                        graph.nodes.set(conditionsId, {
                            type: 'conditions',
                            label: template.conditions,
                            icon: icons.Conditions,
                            classNames: ''
                        });
                        graph.edges.set({v: sensorId, w: conditionsId}, {classNames: this.edgeClassNames(sensorId)});
                        graph.edges.set({v: conditionsId, w: triggerId}, {classNames: this.edgeClassNames(triggerId)});
                    } else {
                        graph.edges.set({v: sensorId, w: triggerId}, {classNames: this.edgeClassNames(triggerId)});
                    }
                });
        });
        return graph;
    }

    private get selected() {
        return this.resource(this.selectedId);
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {
            namespace: this.props.match.params.namespace || '',
            active: {},
            selectedId: this.queryParam('selectedId'),
            tab: this.queryParam('tab'),
            markActivations: !!this.queryParam('markActivations')
        };
    }

    public render() {
        const selected = this.selected;
        const exclude: string[] = [];
        // if the user has selected a specific object then
        if (selected) {
            if (selected.kind === 'Sensor') {
                exclude.push('sensorName');
                if (!!selected.key) {
                    exclude.push('triggerId');
                }
            } else {
                exclude.push('eventSourceName');
                if (!!selected.key) {
                    exclude.push('eventSourceName');
                    exclude.push('eventName');
                }
            }
        }
        const log = (e: any) =>
            Object.entries(e)
                .filter(([key]) => !exclude.includes(key))
                .map(([key, value]) => key + '=' + value)
                .join(', ') + '\n';
        return (
            <Page
                title='Namespace'
                toolbar={{
                    breadcrumbs: [{title: 'Namespaces', path: uiUrl('namespaces')}],
                    actionMenu: {
                        items: [
                            {
                                action: () => (this.markActivations = !this.markActivations),
                                iconClassName: this.markActivations ? 'fa fa-toggle-on' : 'fa fa-toggle-off',
                                title: 'Mark activations'
                            }
                        ]
                    },
                    tools: [<NamespaceFilter key='namespace-filter' value={this.namespace} onChange={namespace => (this.namespace = namespace)} />]
                }}>
                {this.renderGraph()}
                <SlidingPanel isShown={!!selected} onClose={() => (this.selectedId = null)}>
                    {!!selected && (
                        <div>
                            <h4>
                                {selected.kind}/{selected.name} {selected.key}
                            </h4>
                            <Tabs
                                navTransparent={true}
                                selectedTabKey={this.tab}
                                onTabSelected={tab => (this.tab = tab)}
                                tabs={[
                                    {
                                        title: 'SUMMARY',
                                        key: 'summary',
                                        content: <ResourceEditor kind={selected.kind} value={selected.value} />
                                    },
                                    {
                                        title: 'LOGS',
                                        key: 'logs',
                                        content: (
                                            <div className='white-box' style={{height: 600}}>
                                                <FullHeightLogsViewer
                                                    source={{
                                                        key: 'logs',
                                                        loadLogs: () =>
                                                            selected.kind === 'Sensor'
                                                                ? services.sensor.sensorsLogs(this.namespace, selected.name, selected.key, '', 50).map(log)
                                                                : services.eventSource.eventSourcesLogs(this.namespace, selected.name, '', selected.key, '', 50).map(log),
                                                        shouldRepeat: () => false
                                                    }}
                                                />
                                            </div>
                                        )
                                    },
                                    {
                                        title: 'EVENTS',
                                        key: 'events',
                                        content: <EventsPanel kind={selected.kind} namespace={selected.namespace} name={selected.name} />
                                    }
                                ]}
                            />
                        </div>
                    )}
                </SlidingPanel>
            </Page>
        );
    }

    public componentDidMount(): void {
        this.fetch();
    }

    public componentWillUnmount() {
        this.stopListWatches();
        this.stopMarkingActivations();
    }

    private resource(i: string) {
        if (!i) {
            return;
        }
        const {type, namespace, name, key} = ID.split(i);
        const kind: Type = ({Trigger: 'Sensor'} as {[key: string]: Type})[type] || type;
        return {
            namespace,
            kind,
            name,
            key,
            value:
                kind === 'EventSource'
                    ? this.state.eventSources.find(x => x.metadata.namespace === namespace && x.metadata.name === name)
                    : this.state.sensors.find(x => x.metadata.namespace === namespace && x.metadata.name === name)
        };
    }

    private renderGraph() {
        const graph = this.graph;
        return (
            <>
                <ErrorNotice error={this.state.error} onReload={() => this.fetch()} style={{margin: 20}} />
                {graph.nodes.size === 0 ? (
                    <ZeroState title='Nothing to show'>
                        <p>
                            Argo Events allow you to trigger workflows, lambadas, and other actions based on receiving events from things like webhooks, message, or a cron
                            schedule.
                        </p>
                        <p>
                            The "mark activations" buttons allows you to see when an entity "activates". This is determined by it writing a log entry. Helpful to debug when things
                            are happening.
                        </p>
                        <p>
                            <a href='https://argoproj.github.io/argo-events/'>Learn more</a>
                        </p>
                    </ZeroState>
                ) : (
                    <GraphPanel
                        graph={graph}
                        selectedNode={this.selectedId}
                        onNodeSelect={selectedId => (this.selectedId = selectedId)}
                        horizontal={true}
                        types={types()}
                        classNames={{Pending: true, Listening: true, Active: true}}
                    />
                )}
            </>
        );
    }

    private saveHistory() {
        const params = [];
        if (this.selectedId) {
            params.push('selectedId=' + this.selectedId);
        }
        if (this.tab) {
            params.push('tab=' + this.tab);
        }
        if (this.markActivations) {
            params.push('markActivations=' + this.markActivations);
        }
        this.appContext.router.history.push(uiUrl(`namespaces/${this.namespace}?${params.join('&')}`));
        Utils.setCurrentNamespace(this.namespace);
    }

    private fetch() {
        this.stopListWatches();
        this.eventSourceListWatch = new ListWatch<EventSource>(
            () => services.eventSource.list(this.namespace),
            resourceVersion => services.eventSource.watch(this.namespace, resourceVersion),
            () => this.setState({error: null}),
            () => this.setState({error: null}),
            eventSources => this.setState({eventSources}),
            error => this.setState({error})
        );
        this.sensorListWatch = new ListWatch<Sensor>(
            () => services.sensor.list(this.namespace),
            resourceVersion => services.sensor.watch(this.namespace, resourceVersion),
            () => this.setState({error: null}),
            () => this.setState({error: null}),
            sensors => this.setState({sensors}),
            error => this.setState({error})
        );
        this.eventSourceListWatch.start();
        this.sensorListWatch.start();
    }

    private stopListWatches() {
        if (this.eventSourceListWatch) {
            this.eventSourceListWatch.stop();
        }
        if (this.sensorListWatch) {
            this.sensorListWatch.stop();
        }
    }

    private stopMarkingActivations() {
        if (this.markActivationsSubscriptions) {
            this.markActivationsSubscriptions.forEach(x => x.unsubscribe());
            this.markActivationsSubscriptions = null;
        }
    }

    private startMarkingActivations() {
        if (this.markActivationsSubscriptions) {
            return;
        }
        this.markActivationsSubscriptions = [
            services.eventSource
                .eventSourcesLogs(this.namespace, '', '', '', 'dispatching', 0)
                .filter(e => !!e && !!e.eventSourceName)
                .subscribe(
                    e =>
                        this.markActive(
                            ID.join({
                                type: 'EventSource',
                                namespace: e.namespace,
                                name: e.eventSourceName,
                                key: e.eventName
                            })
                        ),
                    error => this.setState({error})
                ),
            services.sensor
                .sensorsLogs(this.namespace, '', '', 'successfully processed', 0)
                .filter(e => !!e)
                .subscribe(
                    e => {
                        this.markActive(
                            ID.join({
                                type: 'Sensor',
                                namespace: e.namespace,
                                name: e.sensorName
                            })
                        );
                        if (e.triggerName) {
                            this.markActive(
                                ID.join({
                                    type: 'Trigger',
                                    namespace: e.namespace,
                                    name: e.sensorName,
                                    key: e.triggerName
                                })
                            );
                        }
                    },
                    error => this.setState({error})
                )
        ];
    }

    private markActive(id: string) {
        this.setState(state => {
            clearTimeout(state.active[id]);
            state.active[id] = setTimeout(() => {
                this.setState(s => {
                    delete s.active[id];
                    return {active: s.active};
                });
            }, 2000);
            return {active: state.active};
        });
    }

    private edgeClassNames(id: string) {
        return 'data ' + (!!this.state.active[id] ? ' active' : '');
    }
}

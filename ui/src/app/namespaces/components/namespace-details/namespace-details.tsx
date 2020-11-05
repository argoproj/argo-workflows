import {Page, SlidingPanel, Tabs} from 'argo-ui/src/index';
import * as React from 'react';
import {RouteComponentProps} from 'react-router';
import {Condition, EventSource, kubernetes, Sensor} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {BasePage} from '../../../shared/components/base-page';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {ResourceEditor} from '../../../shared/components/resource-editor/resource-editor';
import {services} from '../../../shared/services';
import {Utils} from '../../../shared/utils';
import {EventsPanel} from '../../../workflows/components/events-panel';
import {FullHeightLogsViewer} from '../../../workflows/components/workflow-logs-viewer/full-height-logs-viewer';
import {Edge, Graph, GraphPanel, Node} from './graph-panel';
import {ID} from './id';

require('../../../workflows/components/workflow-details/workflow-details.scss');

interface State {
    namespace: string;
    selectedId?: string;
    error?: Error;
    resources: { [id: string]: { metadata: kubernetes.ObjectMeta; status?: { conditions?: Condition[] } } };
    touched: { [id: string]: boolean };
}

const icons: { [type: string]: string } = {
    AMQPEvent: 'stream',
    AWSLambdaTrigger: 'microchip',
    ArgoWorkflowTrigger: 'sitemap',
    AzureEventsHubEvent: 'database',
    CalendarEvent: 'clock',
    Conditions: 'filter',
    CustomTrigger: 'puzzle-piece',
    EmitterEvent: 'stream',
    Event: 'circle',
    EventSource: 'circle',
    FileEvent: 'file',
    GenericEvent: 'puzzle-piece',
    GithubEvent: 'code-branch',
    GitlabEvent: 'code-branch',
    HDFSEvent: 'hdd',
    K8STrigger: 'file-code',
    KafkaEvent: 'stream',
    KafkaTrigger: 'stream',
    MinioEvent: 'database',
    NATSEvent: 'stream',
    NATSTrigger: 'stream',
    NSQEvent: 'stream',
    OpenWhiskTrigger: 'microchip',
    PubSubEvent: 'stream',
    PulsarEvent: 'stream',
    RedisEvent: 'th',
    ResourceEvent: 'file-code',
    SNSEvent: 'stream',
    SQSEvent: 'stream',
    Sensor: 'circle',
    SlackEvent: 'comment',
    SlackTrigger: 'comment',
    StorageGridEvent: 'th',
    StripeEvent: 'credit-card',
    Trigger: 'bell',
    WebhookEvent: 'cloud'
};

const eventTypes: { [key: string]: string } = {
    amqp: 'AMQPEvent',
    azureEventsHub: 'AzureEventsHubEvent',
    calendar: 'CalendarEvent',
    emitter: 'EmitterEvent',
    file: 'FileEvent',
    generic: 'GenericEvent',
    github: 'GithubEvent',
    gitlab: 'GitlabEvent',
    hdfs: 'HDFSEvent',
    kafka: 'KafkaEvent',
    minio: 'MinioEvent',
    mqtt: 'MQTTEvent',
    nats: 'NATSEvent',
    nsq: 'NSQEvent',
    pubSub: 'PubSubEvent',
    pulsar: 'PulsarEvent',
    redis: 'RedisEvent',
    resource: 'ResourceEvent',
    slack: 'SlackEvent',
    sns: 'SNSEvent',
    sqs: 'SQSEvent',
    storageGrid: 'StorageGridEvent',
    stripe: 'StripeEvent',
    webhook: 'WebhookEvent'
};

const triggerTypes: { [key: string]: string } = {
    argoWorkflow: 'ArgoWorkflowTrigger',
    awsLambda: 'AWSLambdaTrigger',
    custom: 'CustomTrigger',
    k8s: 'K8STrigger',
    kafka: 'KafkaTrigger',
    nats: 'NATSTrigger',
    openWhisk: 'OpenWhiskTrigger',
    slack: 'SlackTrigger'
};

const phase = (r: { status?: { conditions?: Condition[] } }) => {
    if (!r.status || !r.status.conditions) {
        return '';
    }
    return r.status.conditions.find(c => c.status !== 'True') ? 'Warning' : 'Running';
};

export class NamespaceDetails extends BasePage<RouteComponentProps<any>, State> {
    private get namespace() {
        return this.state.namespace;
    }

    private set namespace(namespace: string) {
        this.fetch(namespace);
    }

    private get graph(): Graph {
        const nodes: Node[] = Object.entries(this.state.resources)
            .filter(([id]) => ID.split(id).type !== 'EventSource')
            .map(([id, resource]) => ({
                id,
                type: ID.split(id).type.toLowerCase(),
                label: resource.metadata.name,
                icon: icons[ID.split(id).type],
                phase: phase(resource),
                touched: !!this.state.touched[id]
            }));

        const edges: Edge[] = [];

        Object.entries(this.state.resources)
            .filter(([eventSourceId]) => ID.split(eventSourceId).type === 'EventSource')
            .forEach(([, eventSource]) => {
                const spec = (eventSource as EventSource).spec;
                Object.entries(spec)
                    .filter(([typeKey]) => ['template', 'service'].indexOf(typeKey) < 0)
                    .forEach(([typeKey, type]) => {
                        Object.keys(type).forEach(key => {
                            const eventId = ID.join({
                                type: 'Event',
                                namespace: eventSource.metadata.namespace,
                                name: eventSource.metadata.name,
                                key
                            });
                            nodes.push({
                                id: eventId,
                                type: typeKey,
                                label: key,
                                phase: phase(eventSource),
                                icon: icons[eventTypes[typeKey] || 'Event']
                            });
                        });
                    });
            });
        Object.entries(this.state.resources)
            .filter(([sensorId]) => ID.split(sensorId).type === 'Sensor')
            .forEach(([sensorId, sensor]) => {
                const spec = (sensor as Sensor).spec;
                spec.dependencies.forEach(d => {
                    const eventId = ID.join({
                        type: 'Event',
                        namespace: sensor.metadata.namespace,
                        name: d.eventSourceName,
                        key: d.eventName
                    });
                    edges.push({x: eventId, y: sensorId});
                });
                spec.triggers
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
                        nodes.push({
                            id: triggerId,
                            label: template.name,
                            type: triggerTypeKey,
                            phase: phase(sensor),
                            icon: icons[triggerTypes[triggerTypeKey] || 'Trigger']
                        });
                        if (template.conditions) {
                            const conditionsId = ID.join({
                                type: 'Conditions',
                                namespace: sensor.metadata.namespace,
                                name: sensor.metadata.name,
                                key: template.conditions
                            });
                            nodes.push({
                                id: conditionsId,
                                label: template.conditions,
                                type: 'conditions',
                                icon: icons['Conditions']
                            });
                            edges.push({x: sensorId, y: conditionsId});
                            edges.push({x: conditionsId, y: triggerId});
                        } else {
                            edges.push({x: sensorId, y: triggerId});
                        }
                    });
            });
        return {nodes, edges};
    }

    private get selected() {
        return this.resource(this.state.selectedId);
    }

    constructor(props: RouteComponentProps<any>, context: any) {
        super(props, context);
        this.state = {namespace: this.props.match.params.namespace || '', resources: {}, touched: {}};
    }

    public render() {
        const selected = this.selected;
        return (
            <Page
                title='Namespace'
                toolbar={{
                    breadcrumbs: [{title: 'Namespaces', path: uiUrl('namespaces')}],
                    tools: [<NamespaceFilter key='namespace-filter' value={this.namespace}
                                             onChange={namespace => (this.namespace = namespace)}/>]
                }}>
                <div className='argo-container'>{this.renderGraph()}</div>
                <SlidingPanel isShown={!!selected} onClose={() => this.setState({selectedId: null})}>
                    {!!selected && (
                        <>
                            <h4>
                                {selected.kind} / {selected.name}
                            </h4>
                            <Tabs
                                navTransparent={true}
                                tabs={[
                                    {
                                        title: 'SUMMARY',
                                        key: 'summary',
                                        content: <ResourceEditor readonly={true} kind={selected.kind}
                                                                 value={selected.value}/>
                                    },
                                    {
                                        title: 'EVENTS',
                                        key: 'events',
                                        content: <EventsPanel kind={selected.kind} namespace={selected.namespace}
                                                              name={selected.name}/>
                                    },
                                    {
                                        title: 'LOGS',
                                        key: 'logs',
                                        content: (
                                            <div className='white-box' style={{height: 400}}>
                                                <FullHeightLogsViewer
                                                    source={{
                                                        key: 'logs',
                                                        loadLogs: () =>
                                                            this.selected.kind === 'Sensor'
                                                                ? services.sensor.sensorsLogs(this.namespace).map(e => e.content + '\n')
                                                                : services.eventSource.eventSourcesLogs(this.namespace).map(e => e.content + '\n'),
                                                        shouldRepeat: () => false
                                                    }}
                                                />
                                            </div>
                                        )
                                    }
                                ]}
                            />
                        </>
                    )}
                </SlidingPanel>
            </Page>
        );
    }

    public componentDidMount(): void {
        this.fetch(this.namespace);
    }

    private resource(i: string) {
        if (!i) {
            return;
        }
        const {type, namespace, name} = ID.split(i);
        const kind = ({Event: 'EventSource', Trigger: 'Sensor'} as { [key: string]: string })[type] || type;
        return {namespace, kind, name, value: this.state.resources[ID.join({type: kind, namespace, name})]};
    }

    private renderGraph() {
        if (this.state.error) {
            return <ErrorNotice error={this.state.error} onReload={() => this.fetch(this.namespace)}/>;
        }
        return (
            <div style={{textAlign: 'center'}}>
                <GraphPanel graph={this.graph} onSelect={selectedId => this.setState({selectedId})}/>
            </div>
        );
    }

    private fetch(namespace: string) {
        const updateResources = (s: State, type: string, list: { items: { metadata: kubernetes.ObjectMeta }[] }) => {
            (list.items || []).forEach(v => {
                s.resources[ID.join({type, namespace: v.metadata.namespace, name: v.metadata.name})] = v;
            });
            return {resources: s.resources};
        };
        this.setState({resources: {}}, () => {
            Promise.all([
                services.eventSource.list(namespace).then(list => this.setState(s => updateResources(s, 'EventSource', list))),
                services.sensor.list(namespace).then(list => this.setState(s => updateResources(s, 'Sensor', list)))
            ])
                .then(() =>
                    this.setState({error: null, namespace}, () => {
                        this.appContext.router.history.push(uiUrl(`namespaces/${namespace}`));
                        Utils.setCurrentNamespace(namespace);
                    })
                )
                .then(() => {
                    services.sensor.sensorsLogs(namespace, 0).subscribe(
                        e => {
                            const id = ID.join({type: 'Sensor', namespace, name: e.sensorName});
                            this.setState(
                                s => {
                                    s.touched[id] = true;
                                    return {touched: s.touched};
                                },
                                () => {
                                    setTimeout(() => {
                                        this.setState(s => {
                                            delete s.touched[id];
                                            return {touched: s.touched};
                                        });
                                    }, 10000);
                                }
                            );
                        },
                        error => this.setState({error})
                    );
                })
                .catch(error => this.setState({error}));
        });
    }
}

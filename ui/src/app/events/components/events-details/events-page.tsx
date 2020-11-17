import {Page, SlidingPanel, Tabs} from 'argo-ui';
import {useEffect, useState} from 'react';
import React = require('react');
import {RouteComponentProps} from 'react-router-dom';
import {Observable} from 'rxjs';
import {Condition, kubernetes} from '../../../../models';
import {EventSource, eventSourceTypes} from '../../../../models/event-source';
import {Sensor, triggerTypes} from '../../../../models/sensor';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {GraphPanel} from '../../../shared/components/graph/graph-panel';
import {Graph, Node} from '../../../shared/components/graph/types';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {ResourceEditor} from '../../../shared/components/resource-editor/resource-editor';
import {ZeroState} from '../../../shared/components/zero-state';
import {toHistory} from '../../../shared/history';
import {ListWatch} from '../../../shared/list-watch';
import {services} from '../../../shared/services';
import {EventsPanel} from '../../../workflows/components/events-panel';
import {FullHeightLogsViewer} from '../../../workflows/components/workflow-logs-viewer/full-height-logs-viewer';
import {icons} from './icons';
import {ID} from './id';

require('./event-page.scss');

const status = (r: {status?: {conditions?: Condition[]}}) => {
    if (!r.status || !r.status.conditions) {
        return '';
    }
    return !!r.status.conditions.find(c => c.status !== 'True') ? 'Pending' : 'Listening';
};

const types = (() => {
    const v: {[label: string]: boolean} = {sensor: true, conditions: true};
    Object.keys(eventSourceTypes)
        .concat(Object.keys(triggerTypes))
        .forEach(label => (v[label] = true));
    return v;
})();

const buildGraph = (eventSources: EventSource[], sensors: Sensor[], flow: {[id: string]: any}) => {
    const edgeClassNames = (id: Node) => (!!flow[id] ? 'flow' : '');
    const graph = new Graph();

    (eventSources || []).forEach(eventSource => {
        Object.entries(eventSource.spec)
            .filter(([typeKey]) => ['template', 'service'].indexOf(typeKey) < 0)
            .forEach(([typeKey, type]) => {
                Object.keys(type).forEach(key => {
                    const eventId = ID.join('EventSource', eventSource.metadata.namespace, eventSource.metadata.name, key);
                    graph.nodes.set(eventId, {type: typeKey, label: key, classNames: status(eventSource), icon: icons[eventSourceTypes[typeKey] + 'EventSource']});
                });
            });
    });

    (sensors || []).forEach(sensor => {
        const sensorId = ID.join('Sensor', sensor.metadata.namespace, sensor.metadata.name);
        graph.nodes.set(sensorId, {type: 'sensor', label: sensor.metadata.name, icon: icons.Sensor, classNames: status(sensor)});
        (sensor.spec.dependencies || []).forEach(d => {
            const eventId = ID.join('EventSource', sensor.metadata.namespace, d.eventSourceName, d.eventName);
            graph.edges.set({v: eventId, w: sensorId}, {label: d.name, classNames: edgeClassNames(eventId)});
        });
        (sensor.spec.triggers || [])
            .map(t => t.template)
            .filter(template => template)
            .forEach(template => {
                const triggerTypeKey = Object.keys(template).filter(t => ['name', 'conditions'].indexOf(t) === -1)[0];
                const triggerId = ID.join('Trigger', sensor.metadata.namespace, sensor.metadata.name, template.name);
                graph.nodes.set(triggerId, {
                    label: template.name,
                    type: triggerTypeKey,
                    classNames: status(sensor),
                    icon: icons[triggerTypes[triggerTypeKey] + 'Trigger']
                });
                if (template.conditions) {
                    const conditionsId = ID.join('Conditions', sensor.metadata.namespace, sensor.metadata.name, template.conditions);
                    graph.nodes.set(conditionsId, {
                        type: 'conditions',
                        label: template.conditions,
                        icon: icons.Conditions,
                        classNames: ''
                    });
                    graph.edges.set({v: sensorId, w: conditionsId}, {classNames: edgeClassNames(sensorId)});
                    graph.edges.set({v: conditionsId, w: triggerId}, {classNames: edgeClassNames(triggerId)});
                } else {
                    graph.edges.set({v: sensorId, w: triggerId}, {classNames: edgeClassNames(triggerId)});
                }
            });
    });
    return graph;
};

export const EventsPage = (props: RouteComponentProps<any>) => {
    // boiler-plate
    const {match, location, history} = props;
    const queryParams = new URLSearchParams(location.search);

    // state for URL and query parameters
    const [namespace, setNamespace] = useState(match.params.namespace);
    const [showFlow, setShowFlow] = useState(queryParams.get('showFlow') === 'true');
    const [selectedNode, setSelectedNode] = useState<Node>(queryParams.get('selectedNode'));
    const [tab, setTab] = useState<Node>(queryParams.get('tab'));
    useEffect(() => history.push(toHistory('events/{namespace}', {namespace, showFlow, selectedNode, tab})), [namespace, showFlow, selectedNode, tab]);

    // internal state
    const [error, setError] = useState<Error>();
    const [eventSources, setEventSources] = useState<EventSource[]>();
    const [sensors, setSensors] = useState<Sensor[]>();
    const [flow, setFlow] = useState<{[id: string]: any}>({}); // event flowing?

    // when namespace changes, we must reload
    useEffect(() => {
        const listWatch = new ListWatch<EventSource>(
            () => services.eventSource.list(namespace),
            resourceVersion => services.eventSource.watch(namespace, resourceVersion),
            () => setError(null),
            () => setError(null),
            setEventSources,
            setError
        );
        listWatch.start();
        return () => listWatch.stop();
    }, [namespace]);
    useEffect(() => {
        const listWatch = new ListWatch<Sensor>(
            () => services.sensor.list(namespace),
            resourceVersion => services.sensor.watch(namespace, resourceVersion),
            () => setError(null),
            () => setError(null),
            setSensors,
            setError
        );
        listWatch.start();
        return () => listWatch.stop();
    }, [namespace]);

    // follow logs and mark flow
    const markFlowing = (id: Node) => {
        setFlow(newFlow => {
            clearTimeout(newFlow[id]);
            newFlow[id] = setTimeout(() => {
                setFlow(evenNewerFlow => {
                    delete evenNewerFlow[id];
                    return Object.assign({}, evenNewerFlow); // Object.assign work-around to make sure state updates
                });
            }, 2000);
            return Object.assign({}, newFlow);
        });
    };
    useEffect(() => {
        if (!showFlow) {
            return;
        }
        const sub = services.eventSource
            .eventSourcesLogs(namespace, '', '', '', 'dispatching', 0)
            .filter(e => !!e && !!e.eventSourceName)
            .subscribe(e => markFlowing(ID.join('EventSource', e.namespace, e.eventSourceName, e.eventName)), setError);
        return () => sub.unsubscribe();
    }, [namespace, showFlow]);
    useEffect(() => {
        if (!showFlow) {
            return;
        }
        const sub = services.sensor
            .sensorsLogs(namespace, '', '', 'successfully processed', 0)
            .filter(e => !!e)
            .subscribe(e => {
                markFlowing(ID.join('Sensor', e.namespace, e.sensorName));
                if (e.triggerName) {
                    markFlowing(ID.join('Trigger', e.namespace, e.sensorName, e.triggerName));
                }
            }, setError);
        return () => sub.unsubscribe();
    }, [namespace, showFlow]);

    const graph = buildGraph(eventSources, sensors, flow);

    const selected = (() => {
        if (!selectedNode) {
            return;
        }
        const x = ID.split(selectedNode);
        const kind = x.type === 'EventSource' ? 'EventSource' : 'Sensor';
        const resources: {metadata: kubernetes.ObjectMeta}[] = (kind === 'EventSource' ? eventSources : sensors) || [];
        const value = resources.find((y: {metadata: kubernetes.ObjectMeta}) => y.metadata.namespace === x.namespace && y.metadata.name === x.name);
        return {kind, value, ...x};
    })();

    return (
        <Page
            title='Events'
            toolbar={{
                actionMenu: {
                    items: [
                        {
                            action: () => setShowFlow(!showFlow),
                            iconClassName: showFlow ? 'fa fa-toggle-on' : 'fa fa-toggle-off',
                            title: 'Show event-flow'
                        }
                    ]
                },
                tools: [<NamespaceFilter key='namespace-filter' value={namespace} onChange={setNamespace} />]
            }}>
            <ErrorNotice error={error} />
            {graph.nodes.size === 0 ? (
                <ZeroState title='Nothing to show'>
                    <p>Argo Events allow you to trigger workflows, lambadas, and other actions based on receiving events from things like webhooks, message, or a cron schedule.</p>
                    <p>
                        <a href='https://argoproj.github.io/argo-events/'>Learn more</a>
                    </p>
                </ZeroState>
            ) : (
                <>
                    <GraphPanel
                        graph={graph}
                        types={types}
                        classNames={{Pending: true, Listening: true}}
                        horizontal={true}
                        selectedNode={selectedNode}
                        onNodeSelect={setSelectedNode}
                        edgeStrokeWidthMultiple={8}
                    />
                    {showFlow && (
                        <p className='argo-container'>
                            <i className='fa fa-info-circle' /> Event-flow is proxy for events. It is based on the pod logs of the event sources and sensors, so should be treated
                            only as indicative of activity.
                        </p>
                    )}
                </>
            )}
            <SlidingPanel isShown={!!selectedNode} onClose={() => setSelectedNode(null)}>
                {!!selectedNode && (
                    <div>
                        <h4>
                            {selected.kind}/{selected.name}
                        </h4>
                        <h5>{selected.key}</h5>
                        <Tabs
                            navTransparent={true}
                            selectedTabKey={tab}
                            onTabSelected={setTab}
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
                                                        ((selected.kind === 'Sensor'
                                                            ? services.sensor.sensorsLogs(namespace, selected.name, selected.key, '', 50)
                                                            : services.eventSource.eventSourcesLogs(namespace, selected.name, '', selected.key, '', 50)) as Observable<any>)
                                                            .filter(e => !!e)
                                                            .map(
                                                                e =>
                                                                    Object.entries(e)
                                                                        .map(([key, value]) => key + '=' + value)
                                                                        .join(', ') + '\n'
                                                            ),
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
};

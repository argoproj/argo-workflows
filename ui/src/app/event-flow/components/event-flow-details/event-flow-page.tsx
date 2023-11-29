import {Page, SlidingPanel, Tabs} from 'argo-ui';
import {useContext, useEffect, useState} from 'react';
import * as React from 'react';
import {RouteComponentProps} from 'react-router-dom';
import {Observable} from 'rxjs';
import {filter, map} from 'rxjs/operators';
import {kubernetes, Workflow} from '../../../../models';
import {EventSource} from '../../../../models/event-source';
import {Sensor} from '../../../../models/sensor';
import {uiUrl} from '../../../shared/base';
import {Button} from '../../../shared/components/button';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {InfoIcon} from '../../../shared/components/fa-icons';
import {GraphPanel} from '../../../shared/components/graph/graph-panel';
import {Node} from '../../../shared/components/graph/types';
import {Links} from '../../../shared/components/links';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {ResourceEditor} from '../../../shared/components/resource-editor/resource-editor';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {ZeroState} from '../../../shared/components/zero-state';
import {Context} from '../../../shared/context';
import {Footnote} from '../../../shared/footnote';
import {historyUrl} from '../../../shared/history';
import {ListWatch} from '../../../shared/list-watch';
import {RetryObservable} from '../../../shared/retry-observable';
import {services} from '../../../shared/services';
import {useQueryParams} from '../../../shared/use-query-params';
import {Utils} from '../../../shared/utils';
import {EventsPanel} from '../../../workflows/components/events-panel';
import {FullHeightLogsViewer} from '../../../workflows/components/workflow-logs-viewer/full-height-logs-viewer';
import {buildGraph} from './build-graph';
import {genres} from './genres';
import {ID} from './id';

import './event-flow-page.scss';

export function EventFlowPage({history, location, match}: RouteComponentProps<any>) {
    // boiler-plate
    const {navigation} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    // state for URL and query parameters
    const [namespace, setNamespace] = useState(Utils.getNamespace(match.params.namespace) || '');
    const [showFlow, setShowFlow] = useState(queryParams.get('showFlow') === 'true');
    const [showWorkflows, setShowWorkflows] = useState(queryParams.get('showWorkflows') !== 'false');
    const [expanded, setExpanded] = useState(queryParams.get('expanded') === 'true');
    const [selectedNode, setSelectedNode] = useState<Node>(queryParams.get('selectedNode'));
    const [tab, setTab] = useState<Node>(queryParams.get('tab'));

    useEffect(
        useQueryParams(history, p => {
            setShowFlow(p.get('showFlow') === 'true');
            setShowWorkflows(p.get('showWorkflows') === 'true');
            setExpanded(p.get('expanded') === 'true');
            setSelectedNode(p.get('selectedNode'));
            setTab(p.get('tab'));
        }),
        [history]
    );

    useEffect(
        () =>
            history.push(
                historyUrl('event-flow' + (Utils.managedNamespace ? '' : '/{namespace}'), {
                    namespace,
                    showFlow,
                    showWorkflows,
                    expanded,
                    selectedNode,
                    tab
                })
            ),
        [namespace, showFlow, showWorkflows, expanded, expanded, tab]
    );

    // internal state
    const [error, setError] = useState<Error>();
    const [eventSources, setEventSources] = useState<EventSource[]>();
    const [sensors, setSensors] = useState<Sensor[]>();
    const [workflows, setWorkflows] = useState<Workflow[]>();
    const [flow, setFlow] = useState<{[id: string]: {count: number; timeout?: any}}>({}); // event flowing?

    // when namespace changes, we must reload
    useEffect(() => {
        const listWatch = new ListWatch<EventSource>(
            () => services.eventSource.list(namespace),
            () => services.eventSource.watch(namespace),
            () => setError(null),
            () => setError(null),
            items => setEventSources([...items]),
            setError
        );
        listWatch.start();
        return () => listWatch.stop();
    }, [namespace]);
    useEffect(() => {
        const listWatch = new ListWatch<Sensor>(
            () => services.sensor.list(namespace),
            () => services.sensor.watch(namespace),
            () => setError(null),
            () => setError(null),
            items => setSensors([...items]),
            setError
        );
        listWatch.start();
        return () => listWatch.stop();
    }, [namespace]);
    useEffect(() => {
        if (!showWorkflows) {
            setWorkflows(null);
            return;
        }
        const listWatch = new ListWatch<Workflow>(
            () =>
                services.workflows.list(namespace, null, ['events.argoproj.io/sensor', 'events.argoproj.io/trigger'], null, [
                    'metadata',
                    'items.metadata.name',
                    'items.metadata.namespace',
                    'items.metadata.creationTimestamp',
                    'items.metadata.labels'
                ]),
            resourceVersion =>
                services.workflows.watch({
                    namespace,
                    resourceVersion,
                    labels: ['events.argoproj.io/sensor', 'events.argoproj.io/trigger']
                }),
            () => setError(null),
            () => setError(null),
            (items, item, type) => {
                setWorkflows([...items]);
                if (type === 'ADDED') {
                    markFlowing(ID.join('Workflow', item.metadata.namespace, item.metadata.name));
                }
            },
            setError
        );
        listWatch.start();
        return () => listWatch.stop();
    }, [namespace, showWorkflows]);
    // follow logs and mark flow
    const markFlowing = (id: Node) => {
        if (!showFlow) {
            return;
        }
        setError(null);
        setFlow(newFlow => {
            if (!newFlow[id]) {
                newFlow[id] = {count: 0};
            }
            clearTimeout(newFlow[id].timeout);
            newFlow[id].count++;
            newFlow[id].timeout = setTimeout(() => {
                setFlow(evenNewerFlow => {
                    delete evenNewerFlow[id].timeout;
                    return Object.assign({}, evenNewerFlow); // Object.assign work-around to make sure state updates
                });
            }, 3000);
            return Object.assign({}, newFlow);
        });
    };
    useEffect(() => {
        if (!showFlow) {
            return;
        }
        const ro = new RetryObservable(
            () => services.eventSource.eventSourcesLogs(namespace, '', '', '', 'dispatching.*event', 0),
            () => setError(null),
            e => {
                if (e.eventSourceName) {
                    markFlowing(ID.join('EventSource', e.namespace, e.eventSourceName, e.eventName));
                }
            },
            setError
        );
        ro.start();
        return () => ro.stop();
    }, [namespace, showFlow]);
    useEffect(() => {
        if (!showFlow) {
            return;
        }
        const ro = new RetryObservable(
            () => services.sensor.sensorsLogs(namespace, '', '', 'successfully processed', 0),
            () => setError(null),
            e => {
                markFlowing(ID.join('Sensor', e.namespace, e.sensorName));
                if (e.triggerName) {
                    markFlowing(ID.join('Trigger', e.namespace, e.sensorName, e.triggerName));
                }
            },
            setError
        );
        ro.start();
        return () => ro.stop();
    }, [namespace, showFlow]);
    useCollectEvent('openedEventFlow');

    const graph = buildGraph(eventSources, sensors, workflows, flow, expanded);

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

    const emptyGraph = graph.nodes.size === 0;
    return (
        <Page
            title='Event Flow'
            toolbar={{
                breadcrumbs: [
                    {title: 'Event Flow', path: uiUrl('event-flow')},
                    {title: namespace, path: uiUrl('event-flow/' + namespace)}
                ],
                actionMenu: {
                    items: [
                        {
                            action: () => navigation.goto(uiUrl('event-sources/' + namespace + '?sidePanel=true')),
                            iconClassName: 'fa fa-bolt',
                            title: 'Create event source'
                        },
                        {
                            action: () => navigation.goto(uiUrl('sensors/' + namespace + '?sidePanel=true')),
                            iconClassName: 'fa fa-satellite-dish',
                            title: 'Create sensor'
                        },
                        {
                            action: () => setShowFlow(!showFlow),
                            iconClassName: showFlow ? 'fa fa-toggle-on' : 'fa fa-toggle-off',
                            disabled: emptyGraph,
                            title: 'Show event-flow'
                        },
                        {
                            action: () => setShowWorkflows(!showWorkflows),
                            iconClassName: showWorkflows ? 'fa fa-toggle-on' : 'fa fa-toggle-off',
                            disabled: emptyGraph,
                            title: 'Show workflows'
                        },
                        {
                            action: () => setExpanded(!expanded),
                            iconClassName: expanded ? 'fa fa-compress' : 'fa fa-expand',
                            disabled: emptyGraph,
                            title: 'Collapse/expand hidden nodes'
                        }
                    ]
                },
                tools: [<NamespaceFilter key='namespace-filter' value={namespace} onChange={setNamespace} />]
            }}>
            <ErrorNotice error={error} />
            {emptyGraph ? (
                <ZeroState>
                    <p>Argo Events allow you to trigger workflows, lambdas, and other actions when an event such as a webhooks, message, or a cron schedule occurs.</p>
                    <p>
                        <a href='https://argoproj.github.io/argo-events/'>Learn more</a>
                    </p>
                </ZeroState>
            ) : (
                <>
                    <GraphPanel
                        storageScope='events'
                        classNames='events'
                        graph={graph}
                        nodeGenresTitle={'Type'}
                        nodeGenres={genres}
                        nodeClassNamesTitle={'Status'}
                        nodeClassNames={{'': true, 'Pending': true, 'Ready': true, 'Running': true, 'Failed': true, 'Succeeded': true, 'Error': true}}
                        iconShapes={{workflow: 'circle', collapsed: 'circle', conditions: 'circle'}}
                        horizontal={true}
                        selectedNode={selectedNode}
                        onNodeSelect={x => {
                            const id = ID.split(x);
                            if (id.type === 'Workflow') {
                                navigation.goto(uiUrl('workflows/' + id.namespace + '/' + id.name));
                            } else if (id.type === 'Collapsed') {
                                setExpanded(true);
                            } else {
                                setSelectedNode(x);
                            }
                        }}
                    />
                    {showFlow && (
                        <div className='argo-container'>
                            <Footnote>
                                <InfoIcon /> Event-flow is proxy for events. It is based on the pod logs of the event sources and sensors, so should be treated only as indicative
                                of activity.
                            </Footnote>
                        </div>
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
                        <Button
                            outline={true}
                            onClick={() => navigation.goto(uiUrl(`${selected.kind === 'EventSource' ? 'event-sources' : 'sensors'}/${selected.namespace}/${selected.name}`))}>
                            Open/edit
                        </Button>
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
                                        <>
                                            <FullHeightLogsViewer
                                                source={{
                                                    key: 'logs',
                                                    loadLogs: () =>
                                                        ((selected.kind === 'Sensor'
                                                            ? services.sensor.sensorsLogs(namespace, selected.name, selected.key, '', 50)
                                                            : services.eventSource.eventSourcesLogs(namespace, selected.name, '', selected.key, '', 50)) as Observable<any>).pipe(
                                                            filter(e => !!e),
                                                            map(
                                                                e =>
                                                                    Object.entries(e)
                                                                        .map(([key, value]) => key + '=' + value)
                                                                        .join(', ') + '\n'
                                                            )
                                                        ),
                                                    shouldRepeat: () => false
                                                }}
                                            />
                                            {selected.value && <Links scope={selected.kind === 'Sensor' ? 'sensor-logs' : 'event-source-logs'} object={selected.value} />}
                                        </>
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

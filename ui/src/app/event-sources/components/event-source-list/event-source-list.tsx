import {Page, SlidingPanel, Tabs} from 'argo-ui';
import classNames from 'classnames';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {EventSource, kubernetes} from '../../../../models';
import {ID} from '../../../event-flow/components/event-flow-details/id';
import {Utils as EventsUtils} from '../../../sensors/components/utils';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Node} from '../../../shared/components/graph/types';
import {Loading} from '../../../shared/components/loading';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {Timestamp} from '../../../shared/components/timestamp';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {ZeroState} from '../../../shared/components/zero-state';
import {Context} from '../../../shared/context';
import {Footnote} from '../../../shared/footnote';
import {historyUrl} from '../../../shared/history';
import {services} from '../../../shared/services';
import {useQueryParams} from '../../../shared/use-query-params';
import {Utils} from '../../../shared/utils';
import {EventsPanel} from '../../../workflows/components/events-panel';
import {EventSourceCreator} from '../event-source-creator';
import {EventSourceLogsViewer} from '../event-source-log-viewer';

const learnMore = <a href='https://argoproj.github.io/argo-events/concepts/event_source/'>Learn more</a>;

export function EventSourceList({match, location, history}: RouteComponentProps<any>) {
    // boiler-plate
    const queryParams = new URLSearchParams(location.search);
    const {navigation} = useContext(Context);

    // state for URL and query parameters
    const [namespace, setNamespace] = useState(Utils.getNamespace(match.params.namespace) || '');
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel') === 'true');
    const [selectedNode, setSelectedNode] = useState<Node>(queryParams.get('selectedNode'));
    const [tab, setTab] = useState<Node>(queryParams.get('tab'));

    useEffect(
        useQueryParams(history, p => {
            setSidePanel(p.get('sidePanel') === 'true');
            setSelectedNode(p.get('selectedNode'));
            setTab(p.get('tab'));
        }),
        [history]
    );

    useEffect(
        () =>
            history.push(
                historyUrl('event-sources' + (Utils.managedNamespace ? '' : '/{namespace}'), {
                    namespace,
                    sidePanel,
                    selectedNode,
                    tab
                })
            ),
        [namespace, sidePanel, selectedNode, tab]
    );

    // internal state
    const [error, setError] = useState<Error>();
    const [eventSources, setEventSources] = useState<EventSource[]>();

    useEffect(() => {
        services.eventSource
            .list(namespace)
            .then(l => setEventSources(l.items ? l.items : []))
            .then(() => setError(null))
            .catch(setError);
    }, [namespace]);

    const selected = (() => {
        if (!selectedNode) {
            return;
        }
        const x = ID.split(selectedNode);
        const value = (eventSources || []).find((y: {metadata: kubernetes.ObjectMeta}) => y.metadata.namespace === x.namespace && y.metadata.name === x.name);
        return {value, ...x};
    })();

    const loading = !error && !eventSources;
    const zeroState = (eventSources || []).length === 0;

    useCollectEvent('openedEventSourceList');

    return (
        <Page
            title='EventSources'
            toolbar={{
                breadcrumbs: [
                    {title: 'Event Sources', path: uiUrl('event-sources')},
                    {title: namespace, path: uiUrl('event-sources/' + namespace)}
                ],
                actionMenu: {
                    items: [
                        {
                            title: 'Create New EventSource',
                            iconClassName: 'fa fa-plus',
                            action: () => setSidePanel(true)
                        }
                    ]
                },
                tools: [<NamespaceFilter key='namespace-filter' value={namespace} onChange={setNamespace} />]
            }}>
            <ErrorNotice error={error} />
            {loading && <Loading />}
            {zeroState && (
                <ZeroState title='No event sources'>
                    <p>
                        An event source defines what events can be used to trigger actions. Typical event sources are calender (to create events on schedule) GitHub or GitLab (to
                        create events for Git pushes), or MinIO (to create events for file drops). Each event source publishes messages to the event bus so that sensors can listen
                        for them.
                    </p>
                    <p>{learnMore}.</p>
                </ZeroState>
            )}
            {eventSources && eventSources.length > 0 && (
                <>
                    <div className='argo-table-list'>
                        <div className='row argo-table-list__head'>
                            <div className='columns small-1' />
                            <div className='columns small-4'>NAME</div>
                            <div className='columns small-3'>NAMESPACE</div>
                            <div className='columns small-2'>CREATED</div>
                            <div className='columns small-2'>LOGS</div>
                        </div>
                        {eventSources.map(es => (
                            <Link
                                className='row argo-table-list__row'
                                key={`${es.metadata.namespace}/${es.metadata.name}`}
                                to={uiUrl(`event-sources/${es.metadata.namespace}/${es.metadata.name}`)}>
                                <div className='columns small-1'>
                                    <i className={classNames('fa', EventsUtils.statusIconClasses(es.status != null ? es.status.conditions : [], 'fas fa-bolt'))} />
                                </div>
                                <div className='columns small-4'>{es.metadata.name}</div>
                                <div className='columns small-3'>{es.metadata.namespace}</div>
                                <div className='columns small-2'>
                                    <Timestamp date={es.metadata.creationTimestamp} />
                                </div>
                                <div className='columns small-2'>
                                    <div
                                        onClick={() => {
                                            setSelectedNode(`${es.metadata.namespace}/event-sources/${es.metadata.name}`);
                                        }}>
                                        <i className='fa fa-bars' />
                                    </div>
                                </div>
                            </Link>
                        ))}
                    </div>
                    <Footnote>
                        <a onClick={() => navigation.goto(uiUrl('event-flow/' + namespace))}>Show event-flow page</a>
                    </Footnote>
                </>
            )}
            <SlidingPanel isShown={sidePanel} onClose={() => setSidePanel(false)}>
                <EventSourceCreator namespace={namespace} onCreate={es => navigation.goto(uiUrl(`event-sources/${es.metadata.namespace}/${es.metadata.name}`))} />
            </SlidingPanel>
            <SlidingPanel isShown={!!selectedNode} onClose={() => setSelectedNode(null)}>
                {!!selectedNode && (
                    <div>
                        <h4>
                            {selected.name}
                            {selected.key ? '/' + selected.key : ''}
                        </h4>
                        <Tabs
                            navTransparent={true}
                            selectedTabKey={tab}
                            onTabSelected={setTab}
                            tabs={[
                                {
                                    title: 'LOGS',
                                    key: 'logs',
                                    content: <EventSourceLogsViewer namespace={namespace} selectedEvent={selected.key} eventSource={selected.value} onClick={setSelectedNode} />
                                },
                                {
                                    title: 'EVENTS',
                                    key: 'events',
                                    content: <EventsPanel kind='EventSources' namespace={selected.namespace} name={selected.name} />
                                }
                            ]}
                        />
                    </div>
                )}
            </SlidingPanel>
        </Page>
    );
}

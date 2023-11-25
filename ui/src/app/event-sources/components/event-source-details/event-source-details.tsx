import {NotificationType, Page} from 'argo-ui';
import {SlidingPanel, Tabs} from 'argo-ui/src/index';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {EventSource} from '../../../../models';
import {ID} from '../../../event-flow/components/event-flow-details/id';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {Context} from '../../../shared/context';
import {historyUrl} from '../../../shared/history';
import {services} from '../../../shared/services';
import {useQueryParams} from '../../../shared/use-query-params';
import {EventsPanel} from '../../../workflows/components/events-panel';
import {EventSourceEditor} from '../event-source-editor';
import {EventSourceLogsViewer} from '../event-source-log-viewer';

export function EventSourceDetails({history, location, match}: RouteComponentProps<any>) {
    // boiler-plate
    const {notifications, navigation, popup} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    // state for URL and query parameters
    const namespace = match.params.namespace;
    const name = match.params.name;
    const [tab, setTab] = useState<string>(queryParams.get('tab'));
    const [selectedNode, setSelectedNode] = useState<string>(queryParams.get('selectedNode'));

    useEffect(
        useQueryParams(history, p => {
            setTab(p.get('tab'));
            setSelectedNode(p.get('selectedNode'));
        }),
        [history]
    );

    useEffect(
        () =>
            history.push(
                historyUrl('event-sources/{namespace}/{name}', {
                    namespace,
                    name,
                    tab,
                    selectedNode
                })
            ),
        [namespace, name, tab, selectedNode]
    );

    const [edited, setEdited] = useState(false);
    const [error, setError] = useState<Error>();
    const [eventSource, setEventSource] = useState<EventSource>();

    const selected = (() => {
        if (!selectedNode) {
            return;
        }
        const x = ID.split(selectedNode);
        const value = eventSource;
        return {value, ...x};
    })();

    useEffect(() => {
        (async () => {
            try {
                const newEventSource = await services.eventSource.get(name, namespace);
                setEventSource(newEventSource);
                setEdited(false); // set back to false
                setError(null);
            } catch (err) {
                setError(err);
            }
        })();
    }, [name, namespace]);

    useEffect(() => setEdited(true), [eventSource]);

    useCollectEvent('openedEventSourceDetails');

    return (
        <Page
            title='Event Source Details'
            toolbar={{
                breadcrumbs: [
                    {title: 'Event Source', path: uiUrl('event-sources')},
                    {title: namespace, path: uiUrl('event-sources/' + namespace)},
                    {title: name, path: uiUrl('event-sources/' + namespace + '/' + name)}
                ],
                actionMenu: {
                    items: [
                        {
                            title: 'Update',
                            iconClassName: 'fa fa-save',
                            disabled: !edited,
                            action: () =>
                                services.eventSource
                                    .update(eventSource, name, namespace)
                                    .then(setEventSource)
                                    .then(() =>
                                        notifications.show({
                                            content: 'Updated',
                                            type: NotificationType.Success
                                        })
                                    )
                                    .then(() => setEdited(false))
                                    .then(() => setError(null))
                                    .catch(setError)
                        },
                        {
                            title: 'Delete',
                            iconClassName: 'fa fa-trash',
                            disabled: edited,
                            action: () => {
                                popup.confirm('Confirm', 'Are you sure you want to delete this event source?\nThere is no undo.').then(yes => {
                                    if (yes) {
                                        services.eventSource
                                            .delete(name, namespace)
                                            .then(() => navigation.goto(uiUrl('event-sources/' + namespace)))
                                            .then(() => setError(null))
                                            .catch(setError);
                                    }
                                });
                            }
                        },
                        {
                            title: 'Logs',
                            iconClassName: 'fa fa-bars',
                            disabled: false,
                            action: () => {
                                setSelectedNode(`${namespace}/event-sources/${eventSource.metadata.name}`);
                            }
                        }
                    ]
                }
            }}>
            <>
                <ErrorNotice error={error} />
                {!eventSource ? (
                    <Loading />
                ) : (
                    <EventSourceEditor eventSource={eventSource} onChange={setEventSource} onError={setError} onTabSelected={setTab} selectedTabKey={tab} />
                )}
            </>
            <SlidingPanel isShown={!!selected} onClose={() => setSelectedNode(null)}>
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

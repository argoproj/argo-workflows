import * as React from 'react';
import {useEffect, useState} from 'react';
import {Observable} from 'rxjs';
import {filter, map} from 'rxjs/operators';
import {EventSource} from '../../../models';
import {ErrorNotice} from '../../shared/components/error-notice';
import {Links} from '../../shared/components/links';
import {services} from '../../shared/services';
import {FullHeightLogsViewer} from '../../workflows/components/workflow-logs-viewer/full-height-logs-viewer';

function identity<T>(value: T) {
    return () => value;
}

export function EventSourceLogsViewer({
    namespace,
    selectedEvent: selectedEvent,
    eventSource,
    onClick
}: {
    namespace: string;
    selectedEvent: string;
    eventSource: EventSource;
    onClick: (selectedNode: string) => void;
}) {
    const [error, setError] = useState<Error>();
    const [logsObservable, setLogsObservable] = useState<Observable<string>>();
    const [logLoaded, setLogLoaded] = useState(false);
    useEffect(() => {
        if (!eventSource) {
            return;
        }
        const parts = selectedEvent != null ? selectedEvent.split('-') : ['', ''];
        setError(null);
        setLogLoaded(false);
        const source = services.eventSource.eventSourcesLogs(namespace, eventSource.metadata.name, parts[0], parts[1], '', 50).pipe(
            filter(e => !!e),
            map(
                e =>
                    Object.entries(e)
                        .map(([key, value]) => key + '=' + value)
                        .join(', ') + '\n'
            )
        );
        const subscription = source.subscribe(() => setLogLoaded(true), setError);
        setLogsObservable(source);
        return () => subscription.unsubscribe();
    }, [namespace, eventSource, selectedEvent]);

    return (
        <div>
            <div className='row'>
                <div className='columns small-3 medium-2'>
                    <p>Events</p>
                    {error && <ErrorNotice error={error} />}
                    <div style={{marginBottom: '1em'}}>
                        <div
                            key='all'
                            onClick={() => {
                                onClick(`${eventSource.metadata.namespace}/event-sources/${eventSource.metadata.name}`);
                            }}>
                            {!selectedEvent && <i className='fa fa-angle-right' />}
                            {!!selectedEvent && <span>&nbsp;&nbsp;</span>}
                            <a>
                                <span title='all'>all</span>
                            </a>
                        </div>
                        {!!eventSource &&
                            Object.entries(eventSource.spec).map(([type, value]) => (
                                <div key={type}>
                                    <span title={type}>&nbsp;{type}</span>
                                    {Object.entries(value).map(([name]) => (
                                        <div
                                            key={`${type}-${name}`}
                                            onClick={() => {
                                                onClick(`${namespace}/event-sources/${eventSource.metadata.name}/${type}-${name}`);
                                            }}>
                                            {selectedEvent === type + '-' + name && <i className='fa fa-angle-right' />}
                                            {selectedEvent !== type + '-' + name && <span>&nbsp;&nbsp;</span>}
                                            <a>
                                                <span title={name}>&nbsp;&nbsp;{name}</span>
                                            </a>
                                        </div>
                                    ))}
                                </div>
                            ))}
                    </div>
                </div>
                <div className='columns small-9 medium-10'>
                    {!logLoaded ? (
                        <p>
                            <i className='fa fa-circle-notch fa-spin' /> Waiting for data...
                        </p>
                    ) : (
                        <FullHeightLogsViewer
                            source={{
                                key: 'logs',
                                loadLogs: identity(logsObservable),
                                shouldRepeat: () => false
                            }}
                        />
                    )}
                    <Links scope='event-source-logs' object={eventSource} />
                </div>
            </div>
        </div>
    );
}

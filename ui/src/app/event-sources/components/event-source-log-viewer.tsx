import * as React from 'react';
import {useEffect, useState} from 'react';
import {Observable} from 'rxjs';
import {EventSource} from '../../../models';
import {ErrorNotice} from '../../shared/components/error-notice';
import {services} from '../../shared/services';
import {FullHeightLogsViewer} from '../../workflows/components/workflow-logs-viewer/full-height-logs-viewer';

function identity<T>(value: T) {
    return () => value;
}

export const EventSourceLogsViewer = ({
    namespace,
    selectedEvent: selectedEvent,
    eventSource,
    onClick
}: {
    namespace: string;
    selectedEvent: string;
    eventSource: EventSource;
    onClick: (selectedNode: string) => void;
}) => {
    const [error, setError] = useState<Error>();
    const [logsObservable, setLogsObservable] = useState<Observable<string>>();
    const [logLoaded, setLogLoaded] = useState(false);
    const [eventType, setEventType] = useState('');
    const [eventName, setEventName] = useState('');
    useEffect(() => {
        if (!eventSource) {
            return;
        }
        setError(null);
        setLogLoaded(false);

        if (selectedEvent != null) {
            const parts = selectedEvent.split('-');
            setEventType(parts[0]);
            setEventName(parts[1]);
        }
        const source = services.eventSource
            .eventSourcesLogs(namespace, eventSource.metadata.name, eventType, eventName, '', 50)
            .filter(e => !!e)
            .map(
                e =>
                    Object.entries(e)
                        .map(([key, value]) => key + '=' + value)
                        .join(', ') + '\n'
            )
            .publishReplay()
            .refCount();
        const subscription = source.subscribe(() => setLogLoaded(true), setError);
        setLogsObservable(source);
        return () => subscription.unsubscribe();
    }, [namespace, eventSource, selectedEvent]);

    // @ts-ignore
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
                            Object.entries(eventSource.spec).map(([key, value]) => (
                                <div key={{key}}>
                                    <span title={key}>&nbsp;{key}</span>
                                    {Object.entries(value).map(([name, eventValue]) => (
                                        <div
                                            key={name}
                                            onClick={() => {
                                                onClick(`${namespace}/event-sources/${eventSource.metadata.name}/${key}-${name}`);
                                            }}>
                                            {selectedEvent === key + '-' + name && <i className='fa fa-angle-right' />}
                                            {selectedEvent !== key + '-' + name && <span>&nbsp;&nbsp;</span>}
                                            <a>
                                                <span title={name}>&nbsp;&nbsp;{name}</span>
                                            </a>
                                        </div>
                                    ))}
                                </div>
                            ))}
                    </div>
                </div>
                <div className='columns small-9 medium-10' style={{height: 600}}>
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
                </div>
            </div>
        </div>
    );
};

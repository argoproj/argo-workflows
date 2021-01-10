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
    selectedTrigger,
    eventSource,
    onClick
}: {
    namespace: string;
    selectedTrigger: string;
    eventSource: EventSource;
    onClick: (selectedNode: string) => void;
}) => {
    const [error, setError] = useState<Error>();
    const [logsObservable, setLogsObservable] = useState<Observable<string>>();
    const [logLoaded, setLogLoaded] = useState(false);

    useEffect(() => {
        if (!eventSource) {
            return;
        }
        setError(null);
        setLogLoaded(false);
        const source = services.eventSource
            .eventSourcesLogs(namespace, eventSource.metadata.name, 'calender', selectedTrigger, `50`)
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
    }, [namespace, eventSource, selectedTrigger]);

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
                                onClick(`${namespace}/event-sources/${eventSource.metadata.name}`);
                            }}>
                            {!selectedTrigger && <i className='fa fa-angle-right' />}
                            {!!selectedTrigger && <span>&nbsp;&nbsp;</span>}
                            <a>
                                <span title='all'>all</span>
                            </a>
                        </div>
                        {!!eventSource &&
                        Object.entries(eventSource.spec.calendar).map(([key, value]) => (
                                <div
                                    key={key}
                                    onClick={() => {
                                        onClick(`${namespace}/event-sources/${eventSource.metadata.name}/${key}`);
                                    }}>
                                    {selectedTrigger === key && <i className='fa fa-angle-right' />}
                                    {selectedTrigger !== key && <span>&nbsp;&nbsp;</span>}
                                    <a>
                                        <span title={key}>{key}</span>
                                    </a>
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

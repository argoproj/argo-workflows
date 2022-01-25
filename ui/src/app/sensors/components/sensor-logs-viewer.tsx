import * as React from 'react';
import {useEffect, useState} from 'react';
import {Observable} from 'rxjs';
import {filter, map, publishReplay, refCount} from 'rxjs/operators';
import {Sensor} from '../../../models';
import {ErrorNotice} from '../../shared/components/error-notice';
import {Links} from '../../shared/components/links';
import {services} from '../../shared/services';
import {FullHeightLogsViewer} from '../../workflows/components/workflow-logs-viewer/full-height-logs-viewer';

function identity<T>(value: T) {
    return () => value;
}

export const SensorLogsViewer = ({
    namespace,
    selectedTrigger,
    sensor,
    onClick
}: {
    namespace: string;
    selectedTrigger: string;
    sensor: Sensor;
    onClick: (selectedNode: string) => void;
}) => {
    const [error, setError] = useState<Error>();
    const [logsObservable, setLogsObservable] = useState<Observable<string>>();
    const [logLoaded, setLogLoaded] = useState(false);

    useEffect(() => {
        if (!sensor) {
            return;
        }
        setError(null);
        setLogLoaded(false);
        const source = services.sensor.sensorsLogs(namespace, sensor.metadata.name, selectedTrigger, '', 50).pipe(
            filter(e => !!e),
            map(
                e =>
                    Object.entries(e)
                        .map(([key, value]) => key + '=' + value)
                        .join(', ') + '\n'
            ),
            publishReplay(),
            refCount()
        );
        const subscription = source.subscribe(() => setLogLoaded(true), setError);
        setLogsObservable(source);
        return () => subscription.unsubscribe();
    }, [namespace, sensor, selectedTrigger]);

    return (
        <div>
            <div className='row'>
                <div className='columns small-3 medium-2'>
                    <p>Triggers</p>
                    <div style={{marginBottom: '1em'}}>
                        <div
                            key='all'
                            onClick={() => {
                                onClick(`${namespace}/Sensor/${sensor.metadata.name}`);
                            }}>
                            {!selectedTrigger && <i className='fa fa-angle-right' />}
                            {!!selectedTrigger && <span>&nbsp;&nbsp;</span>}
                            <a>
                                <span title='all'>all</span>
                            </a>
                        </div>
                        {!!sensor &&
                            sensor.spec.triggers.map(x => (
                                <div
                                    key={x.template.name}
                                    onClick={() => {
                                        onClick(`${namespace}/Trigger/${sensor.metadata.name}/${x.template.name}`);
                                    }}>
                                    {selectedTrigger === x.template.name && <i className='fa fa-angle-right' />}
                                    {selectedTrigger !== x.template.name && <span>&nbsp;&nbsp;</span>}
                                    <a>
                                        <span title={x.template.name}>{x.template.name}</span>
                                    </a>
                                </div>
                            ))}
                    </div>
                    {error && <ErrorNotice error={error} />}
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
                    <Links scope='sensor-logs' object={sensor} />
                </div>
            </div>
        </div>
    );
};

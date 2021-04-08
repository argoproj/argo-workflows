import {Select} from 'argo-ui';
import * as React from 'react';
import {useEffect, useState} from 'react';
import {Observable} from 'rxjs';
import {ErrorNotice} from '../../shared/components/error-notice';
import {services} from '../../shared/services';
import {FullHeightLogsViewer} from '../../workflows/components/workflow-logs-viewer/full-height-logs-viewer';

function identity<T>(value: T) {
    return () => value;
}

export const PipelineLogsViewer = ({namespace, pipelineName, stepName}: {namespace: string; pipelineName: string; stepName: string}) => {
    const [container, setContainer] = useState<string>('main');
    const [error, setError] = useState<Error>();
    const [logsObservable, setLogsObservable] = useState<Observable<string>>();
    const [logLoaded, setLogLoaded] = useState(false);

    useEffect(() => {
        setError(null);
        setLogLoaded(false);
        const source = services.pipeline
            .pipelineLogs(namespace, pipelineName, stepName, container, 50)
            .filter(e => !!e)
            .map(e => e.msg + '\n')
            .publishReplay()
            .refCount();
        const subscription = source.subscribe(() => setLogLoaded(true), setError);
        setLogsObservable(source);
        return () => subscription.unsubscribe();
    }, [namespace, pipelineName, stepName, container]);

    return (
        <div>
            <div className='row'>
                <div className='columns small-3 medium-2'>
                    <p>Container</p>
                    <div style={{marginBottom: '1em'}}>
                        <Select options={['main', 'sidecar', 'init']} onChange={option => setContainer(option.value)} value={container} />
                    </div>
                    <ErrorNotice error={error} />
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
                </div>
            </div>
        </div>
    );
};

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
    const [container, setContainer] = useState('main');
    const [tailLines, setTailLines] = useState(50);
    const [error, setError] = useState<Error>();
    const [grep, setGrep] = useState('');
    const [logsObservable, setLogsObservable] = useState<Observable<string>>();
    const [logLoaded, setLogLoaded] = useState(false);
    // filter allows us to introduce a short delay, before we actually change grep
    const [filter, setFilter] = useState('');
    useEffect(() => {
        const x = setTimeout(() => setGrep(filter), 1000);
        return () => clearTimeout(x);
    }, [filter]);

    useEffect(() => {
        setError(null);
        setLogLoaded(false);
        const source = services.pipeline
            .pipelineLogs(namespace, pipelineName, stepName, container, grep, tailLines)
            .filter(e => !!e)
            .map(e => e.msg + '\n')
            // this next line highlights the search term in bold with a yellow background, white text
            .map(x => x.replace(new RegExp(grep, 'g'), y => '\u001b[1m\u001b[43;1m\u001b[37m' + y + '\u001b[0m'))
            .publishReplay()
            .refCount();
        const subscription = source.subscribe(() => setLogLoaded(true), setError);
        setLogsObservable(source);
        return () => subscription.unsubscribe();
    }, [namespace, pipelineName, stepName, container, grep, tailLines]);

    return (
        <div>
            <div>
                {['init', 'main', 'sidecar'].map(x => (
                    <a onClick={() => setContainer(x)} key={x} style={{margin: 10}}>
                        {x === container ? (
                            <b>
                                {' '}
                                <i className='fa fa-angle-right' />
                                {x}
                            </b>
                        ) : (
                            <span>&nbsp;&nbsp;{x}</span>
                        )}
                    </a>
                ))}
                <span className='fa-pull-right'>
                    <i className='fa fa-filter' /> <input type='search' defaultValue={filter} onChange={v => setFilter(v.target.value)} placeholder='Filter (regexp)...' />
                </span>
            </div>
            <ErrorNotice error={error} />
            {!logLoaded ? (
                <div className='log-box'>
                    <i className='fa fa-circle-notch fa-spin' /> Waiting for data...
                </div>
            ) : (
                <FullHeightLogsViewer source={{key: 'logs', loadLogs: identity(logsObservable), shouldRepeat: () => false}} />
            )}
            <div style={{textAlign: 'right'}}>
                <select style={{width: 'auto'}} value={tailLines} onChange={e => setTailLines(parseInt(e.currentTarget.value, 10))}>
                    <option>5</option>
                    <option>50</option>
                    <option>500</option>
                    <option>5000</option>
                </select>
            </div>
        </div>
    );
};

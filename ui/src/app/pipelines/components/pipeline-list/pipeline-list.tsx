import {Page} from 'argo-ui';
import * as React from 'react';
import {useEffect, useState} from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {NodePhase} from '../../../../models';
import {Pipeline} from '../../../../models/pipeline';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {PhaseIcon} from '../../../shared/components/phase-icon';
import {Timestamp} from '../../../shared/components/timestamp';
import {ZeroState} from '../../../shared/components/zero-state';
import {historyUrl} from '../../../shared/history';
import {ListWatch} from '../../../shared/list-watch';
import {services} from '../../../shared/services';

export const PipelineList = ({match, history}: RouteComponentProps<any>) => {
    // state for URL and query parameters
    const [namespace, setNamespace] = useState(match.params.namespace || '');

    useEffect(
        () =>
            history.push(
                historyUrl('pipelines/{namespace}', {
                    namespace
                })
            ),
        [namespace]
    );

    // internal state
    const [error, setError] = useState<Error>();
    const [pipelines, setPipelines] = useState<Pipeline[]>();

    useEffect(() => {
        const lw = new ListWatch<Pipeline>(
            () => services.pipeline.listPipelines(namespace),
            () => services.pipeline.watchPipelines(namespace),
            () => setError(null),
            () => setError(null),
            items => setPipelines([...items]),
            setError
        );
        lw.start();
        return () => lw.stop();
    }, [namespace]);

    const loading = !error && !pipelines;
    const zeroState = (pipelines || []).length === 0;

    return (
        <Page
            title='Pipelines'
            toolbar={{
                breadcrumbs: [
                    {title: 'Pipelines', path: uiUrl('pipelines')},
                    {title: namespace, path: uiUrl('pipelines/' + namespace)}
                ],
                tools: [<NamespaceFilter key='namespace-filter' value={namespace} onChange={setNamespace} />]
            }}>
            <ErrorNotice error={error} />
            {loading && <Loading />}
            {zeroState && (
                <ZeroState title='No pipelines'>
                    <p>Argo Dataflow is a Kubernetes native platform for executing large parallel data-processing pipelines.</p>
                    <p>
                        Each pipeline consists of steps. Each step creates zero or more replicas that load-balance messages from one or more sources (such as a cron, HTTP, Kafka,
                        or NATS Streaming), processes the data, then sink it to one or more sinks (such as a HTTP, log, Kafka, NATS Streaming).
                    </p>
                    <p>
                        Each step can scale horizontally using HPA or based on queue length using built-in scaling rules. Steps can be scaled-to-zero, in which case they
                        periodically briefly scale-to-one to measure queue length in case the pipeline needs to scale back up.
                    </p>
                    <p>
                        <a href='https://github.com/argoproj-labs/argo-dataflow'>Learn more</a>
                    </p>
                </ZeroState>
            )}
            {pipelines && pipelines.length > 0 && (
                <>
                    <div className='argo-table-list'>
                        <div className='row argo-table-list__head'>
                            <div className='columns small-1' />
                            <div className='columns small-2'>NAME</div>
                            <div className='columns small-2'>NAMESPACE</div>
                            <div className='columns small-2'>CREATED</div>
                            <div className='columns small-2'>MESSAGE</div>
                            <div className='columns small-3'>CONDITIONS</div>
                        </div>
                        {pipelines.map(p => (
                            <Link
                                className='row argo-table-list__row'
                                key={`${p.metadata.namespace}/${p.metadata.name}`}
                                to={uiUrl(`pipelines/${p.metadata.namespace}/${p.metadata.name}`)}>
                                <div className='columns small-1'>
                                    <PhaseIcon value={p.status && (p.status.phase as NodePhase)} />
                                </div>
                                <div className='columns small-2'>{p.metadata.name}</div>
                                <div className='columns small-2'>{p.metadata.namespace}</div>
                                <div className='columns small-2'>
                                    <Timestamp date={p.metadata.creationTimestamp} />
                                </div>
                                <div className='columns small-2'>{p.status && p.status.message}</div>
                                <div className='columns small-3'>{p.status && p.status.conditions && p.status.conditions.map(c => c.type).join(',')}</div>
                            </Link>
                        ))}
                    </div>
                </>
            )}
        </Page>
    );
};

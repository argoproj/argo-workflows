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
            {!pipelines ? (
                <Loading />
            ) : pipelines.length === 0 ? (
                <ZeroState title='No pipelines'>
                    <p>A pipeline is something super secret. Shhhh...</p>
                    <p>
                        <a href='https://github.com/argoproj-labs/argo-dataflow'>Learn more</a>.
                    </p>
                </ZeroState>
            ) : (
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

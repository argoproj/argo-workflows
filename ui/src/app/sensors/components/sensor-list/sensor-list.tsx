import {Page, SlidingPanel} from 'argo-ui';
import classNames from 'classnames';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {Link, RouteComponentProps} from 'react-router-dom';
import {kubernetes, Sensor} from '../../../../models';
import {ID} from '../../../event-flow/components/event-flow-details/id';
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
import {SensorCreator} from '../sensor-creator';
import {SensorSidePanel} from '../sensor-side-panel';
import {Utils as EventsUtils} from '../utils';

const learnMore = <a href='https://argoproj.github.io/argo-events/concepts/sensor/'>Learn more</a>;

export const SensorList = ({match, location, history}: RouteComponentProps<any>) => {
    // boiler-plate
    const queryParams = new URLSearchParams(location.search);
    const {navigation} = useContext(Context);

    // state for URL and query parameters
    const [namespace, setNamespace] = useState(Utils.getNamespace(match.params.namespace) || '');
    const [sidePanel, setSidePanel] = useState(queryParams.get('sidePanel') === 'true');
    const [selectedNode, setSelectedNode] = useState<Node>(queryParams.get('selectedNode'));

    useEffect(
        useQueryParams(history, p => {
            setSidePanel(p.get('sidePanel') === 'true');
            setSelectedNode(p.get('selectedNode'));
        }),
        [history]
    );

    useEffect(
        () =>
            history.push(
                historyUrl('sensors' + (Utils.managedNamespace ? '' : '/{namespace}'), {
                    namespace,
                    sidePanel,
                    selectedNode
                })
            ),
        [namespace, sidePanel, selectedNode]
    );

    // internal state
    const [error, setError] = useState<Error>();
    const [sensors, setSensors] = useState<Sensor[]>();

    useEffect(() => {
        services.sensor
            .list(namespace)
            .then(l => setSensors(l.items ? l.items : []))
            .then(() => setError(null))
            .catch(setError);
    }, [namespace]);

    useCollectEvent('openedSensorList');

    const selected = (() => {
        if (!selectedNode) {
            return;
        }
        const x = ID.split(selectedNode);
        const value = (sensors || []).find((y: {metadata: kubernetes.ObjectMeta}) => y.metadata.namespace === x.namespace && y.metadata.name === x.name);
        return {value, ...x};
    })();

    const loading = !error && !sensors;
    const zeroState = (sensors || []).length === 0;

    return (
        <Page
            title='Sensors'
            toolbar={{
                breadcrumbs: [
                    {title: 'Sensors', path: uiUrl('sensors')},
                    {title: namespace, path: uiUrl('sensors/' + namespace)}
                ],
                actionMenu: {
                    items: [
                        {
                            title: 'Create New Sensor',
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
                <ZeroState title='No sensors'>
                    <p>
                        A sensor defines what actions to trigger when certain events occur. Typical events are a Git push, a file dropped into a bucket, or a message on a queue or
                        topic. Typical triggers are start a workflow, creating a Kubernetes resource, or sending a message to another queue or topic. Each sensor listens for events
                        from the event bus, checks to see if they&apos;re the right one, and then triggers some actions.
                    </p>
                    <p>{learnMore}.</p>
                </ZeroState>
            )}
            {sensors && sensors.length > 0 && (
                <>
                    <div className='argo-table-list'>
                        <div className='row argo-table-list__head'>
                            <div className='columns small-1' />
                            <div className='columns small-4'>NAME</div>
                            <div className='columns small-3'>NAMESPACE</div>
                            <div className='columns small-2'>CREATED</div>
                            <div className='columns small-2'>LOGS</div>
                        </div>
                        {sensors.map(s => (
                            <Link
                                className='row argo-table-list__row'
                                key={`${s.metadata.namespace}/${s.metadata.name}`}
                                to={uiUrl(`sensors/${s.metadata.namespace}/${s.metadata.name}`)}>
                                <div className='columns small-1'>
                                    <i
                                        className={classNames('fa', EventsUtils.statusIconClasses(s.status != null ? s.status.conditions : [], 'fa-satellite-dish'))}
                                        aria-hidden='true'
                                    />
                                </div>
                                <div className='columns small-4'>{s.metadata.name}</div>
                                <div className='columns small-3'>{s.metadata.namespace}</div>
                                <div className='columns small-2'>
                                    <Timestamp date={s.metadata.creationTimestamp} />
                                </div>
                                <div className='columns small-2'>
                                    <div
                                        onClick={e => {
                                            e.preventDefault();
                                            setSelectedNode(`${s.metadata.namespace}/Sensor/${s.metadata.name}`);
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
                <SensorCreator namespace={namespace} onCreate={s => navigation.goto(uiUrl(`sensors/${s.metadata.namespace}/${s.metadata.name}`))} />
            </SlidingPanel>
            {!!selectedNode && (
                <SensorSidePanel
                    isShown={!!selectedNode}
                    namespace={namespace}
                    sensor={selected.value}
                    selectedTrigger={selected.key}
                    onTriggerClicked={setSelectedNode}
                    onClose={() => setSelectedNode(null)}
                />
            )}
        </Page>
    );
};

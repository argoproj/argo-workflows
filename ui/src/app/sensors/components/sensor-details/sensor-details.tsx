import {NotificationType, Page} from 'argo-ui';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';

import {Sensor} from '../../../../models';
import {ID} from '../../../event-flow/components/event-flow-details/id';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Node} from '../../../shared/components/graph/types';
import {Loading} from '../../../shared/components/loading';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {Context} from '../../../shared/context';
import {historyUrl} from '../../../shared/history';
import {services} from '../../../shared/services';
import {useQueryParams} from '../../../shared/use-query-params';
import {SensorEditor} from '../sensor-editor';
import {SensorSidePanel} from '../sensor-side-panel';

import '../../../workflows/components/workflow-details/workflow-details.scss';

export function SensorDetails({match, location, history}: RouteComponentProps<any>) {
    // boiler-plate
    const {navigation, notifications, popup} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    const [namespace] = useState(match.params.namespace);
    const [name] = useState(match.params.name);
    const [tab, setTab] = useState<string>(queryParams.get('tab'));

    const [sensor, setSensor] = useState<Sensor>();
    const [edited, setEdited] = useState(false);
    const [selectedLogNode, setSelectedLogNode] = useState<Node>(queryParams.get('selectedLogNode'));
    const [error, setError] = useState<Error>();

    useEffect(
        useQueryParams(history, p => {
            setTab(p.get('tab'));
            setSelectedLogNode(p.get('selectedLogNode'));
        }),
        [history]
    );

    useEffect(
        () =>
            history.push(
                historyUrl('sensors/{namespace}/{name}', {
                    namespace,
                    name,
                    tab,
                    selectedLogNode
                })
            ),
        [namespace, name, tab, selectedLogNode]
    );

    useEffect(() => {
        services.sensor
            .get(name, namespace)
            .then(setSensor)
            .then(() => setEdited(false))
            .then(() => setError(null))
            .catch(setError);
    }, [namespace, name]);

    useEffect(() => setEdited(true), [sensor]);

    useCollectEvent('openedSensorDetails');

    const selected = (() => {
        if (!selectedLogNode) {
            return;
        }
        const x = ID.split(selectedLogNode);
        return {...x};
    })();

    return (
        <Page
            title='Sensor Details'
            toolbar={{
                breadcrumbs: [
                    {title: 'Sensors', path: uiUrl('sensors')},
                    {title: namespace, path: uiUrl('sensors/' + namespace)},
                    {title: name, path: uiUrl('sensors/' + namespace + '/' + name)}
                ],
                actionMenu: {
                    items: [
                        {
                            title: 'Update',
                            iconClassName: 'fa fa-save',
                            disabled: !edited,
                            action: () =>
                                services.sensor
                                    .update(sensor, namespace)
                                    .then(setSensor)
                                    .then(() => notifications.show({content: 'Updated', type: NotificationType.Success}))
                                    .then(() => setEdited(false))
                                    .then(() => setError(null))
                                    .catch(setError)
                        },
                        {
                            title: 'Delete',
                            iconClassName: 'fa fa-trash',
                            disabled: edited,
                            action: () => {
                                popup.confirm('Confirm', `Are you sure you want to delete this sensor object?\nThere is no undo.`).then(yes => {
                                    if (yes) {
                                        services.sensor
                                            .delete(name, namespace)
                                            .then(() => navigation.goto(uiUrl('sensors/' + namespace)))
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
                                setSelectedLogNode(`${namespace}/Sensor/${sensor.metadata.name}`);
                            }
                        }
                    ]
                }
            }}>
            <>
                <ErrorNotice error={error} />
                {!sensor ? <Loading /> : <SensorEditor sensor={sensor} onChange={setSensor} onError={setError} selectedTabKey={tab} onTabSelected={setTab} />}
            </>
            {!!selectedLogNode && (
                <SensorSidePanel
                    isShown={!!selectedLogNode}
                    namespace={namespace}
                    sensor={sensor}
                    selectedTrigger={selected.key}
                    onTriggerClicked={setSelectedLogNode}
                    onClose={() => setSelectedLogNode(null)}
                />
            )}
        </Page>
    );
}

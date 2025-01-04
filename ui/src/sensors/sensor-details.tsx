import {NotificationType} from 'argo-ui/src/components/notifications/notifications';
import {Page} from 'argo-ui/src/components/page/page';
import * as React from 'react';
import {useContext, useEffect, useRef, useState} from 'react';
import {RouteComponentProps} from 'react-router';

import {ID} from '../event-flow/id';
import {uiUrl} from '../shared/base';
import {ErrorNotice} from '../shared/components/error-notice';
import {Node} from '../shared/components/graph/types';
import {Loading} from '../shared/components/loading';
import {Context} from '../shared/context';
import {historyUrl} from '../shared/history';
import {Sensor} from '../shared/models';
import {services} from '../shared/services';
import {useCollectEvent} from '../shared/use-collect-event';
import {useEditableObject} from '../shared/use-editable-object';
import {useQueryParams} from '../shared/use-query-params';
import {SensorEditor} from './sensor-editor';
import {SensorSidePanel} from './sensor-side-panel';

import '../workflows/components/workflow-details/workflow-details.scss';

export function SensorDetails({match, location, history}: RouteComponentProps<any>) {
    // boiler-plate
    const isFirstRender = useRef(true);
    const {navigation, notifications, popup} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    const [namespace] = useState(match.params.namespace);
    const [name] = useState(match.params.name);
    const [tab, setTab] = useState<string>(queryParams.get('tab'));

    const {object: sensor, setObject: setSensor, resetObject: resetSensor, serialization, edited, lang, setLang} = useEditableObject<Sensor>();
    const [selectedLogNode, setSelectedLogNode] = useState<Node>(queryParams.get('selectedLogNode'));
    const [error, setError] = useState<Error>();

    useEffect(
        useQueryParams(history, p => {
            setTab(p.get('tab'));
            setSelectedLogNode(p.get('selectedLogNode'));
        }),
        [history]
    );

    useEffect(() => {
        if (isFirstRender.current) {
            isFirstRender.current = false;
            return;
        }
        history.push(
            historyUrl('sensors/{namespace}/{name}', {
                namespace,
                name,
                tab,
                selectedLogNode
            })
        );
    }, [namespace, name, tab, selectedLogNode]);

    useEffect(() => {
        services.sensor
            .get(name, namespace)
            .then(resetSensor)
            .then(() => setError(null))
            .catch(setError);
    }, [namespace, name]);

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
                                    .then(resetSensor)
                                    .then(() => notifications.show({content: 'Updated', type: NotificationType.Success}))
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
                {!sensor ? (
                    <Loading />
                ) : (
                    <SensorEditor
                        sensor={sensor}
                        serialization={serialization}
                        lang={lang}
                        onChange={setSensor}
                        onLangChange={setLang}
                        onError={setError}
                        selectedTabKey={tab}
                        onTabSelected={setTab}
                    />
                )}
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

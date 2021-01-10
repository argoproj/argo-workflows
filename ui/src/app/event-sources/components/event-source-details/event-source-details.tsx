import {NotificationType, Page} from 'argo-ui';
import * as React from 'react';
import {useContext, useEffect, useState} from 'react';
import {RouteComponentProps} from 'react-router';
import {EventSource} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {Loading} from '../../../shared/components/loading';
import {Context} from '../../../shared/context';
import {historyUrl} from '../../../shared/history';
import {services} from '../../../shared/services';
import {EventSourceEditor} from '../event-source-editor';

export const EventSourceDetails = ({history, location, match}: RouteComponentProps<any>) => {
    // boiler-plate
    const {notifications, navigation} = useContext(Context);
    const queryParams = new URLSearchParams(location.search);

    // state for URL and query parameters
    const namespace = match.params.namespace;
    const name = match.params.name;
    const [tab, setTab] = useState<string>(queryParams.get('tab'));

    useEffect(
        () =>
            history.push(
                historyUrl('event-sources/{namespace}/{name}', {
                    namespace,
                    name,
                    tab
                })
            ),
        [namespace, name, tab]
    );

    const [edited, setEdited] = useState(false);
    const [error, setError] = useState<Error>();
    const [eventSource, setEventSource] = useState<EventSource>();

    useEffect(() => {
        services.eventSource
            .get(name, namespace)
            .then(setEventSource)
            .then(() => setEdited(false)) // set back to false
            .then(() => setError(null))
            .catch(setError);
    }, [name, namespace]);

    useEffect(() => setEdited(true), [eventSource]);

    return (
        <Page
            title='Event Source Details'
            toolbar={{
                breadcrumbs: [
                    {title: 'Event Source', path: uiUrl('event-sources')},
                    {title: namespace, path: uiUrl('event-sources/' + namespace)},
                    {title: name, path: uiUrl('event-sources/' + namespace + '/' + name)}
                ],
                actionMenu: {
                    items: [
                        {
                            title: 'Update',
                            iconClassName: 'fa fa-save',
                            disabled: !edited,
                            action: () =>
                                services.eventSource
                                    .update(eventSource, name, namespace)
                                    .then(setEventSource)
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
                                if (!confirm('Are you sure you want to delete this event source?\nThere is no undo.')) {
                                    return;
                                }
                                services.eventSource
                                    .delete(name, namespace)
                                    .then(() => navigation.goto(uiUrl('event-sources/' + namespace)))
                                    .then(() => setError(null))
                                    .catch(setError);
                            }
                        }
                    ]
                }
            }}>
            <>
                <ErrorNotice error={error} />
                {!eventSource ? (
                    <Loading />
                ) : (
                    <EventSourceEditor eventSource={eventSource} onChange={setEventSource} onError={setError} onTabSelected={setTab} selectedTabKey={tab} />
                )}
            </>
        </Page>
    );
};

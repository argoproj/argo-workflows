import * as React from 'react';
import {useState} from 'react';
import {EventSource} from '../../../models';
import {Button} from '../../shared/components/button';
import {ErrorNotice} from '../../shared/components/error-notice';
import {UploadButton} from '../../shared/components/upload-button';
import {exampleEventSource} from '../../shared/examples';
import {services} from '../../shared/services';
import {Utils} from '../../shared/utils';
import {EventSourceEditor} from './event-source-editor';

export const EventSourceCreator = ({onCreate, namespace}: {namespace: string; onCreate: (eventSource: EventSource) => void}) => {
    const [eventSource, setEventSource] = useState<EventSource>(exampleEventSource(Utils.getNamespace(namespace)));
    const [error, setError] = useState<Error>();
    return (
        <>
            <div>
                <UploadButton onUpload={setEventSource} onError={setError} />
                <Button
                    icon='plus'
                    onClick={() => {
                        services.eventSource
                            .create(eventSource, eventSource.metadata.namespace)
                            .then(onCreate)
                            .catch(setError);
                    }}>
                    Create
                </Button>
            </div>
            <ErrorNotice error={error} />
            <EventSourceEditor eventSource={eventSource} onChange={setEventSource} onError={setError} />
        </>
    );
};

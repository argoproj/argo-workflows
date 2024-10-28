import * as React from 'react';
import {useState} from 'react';

import {Button} from '../shared/components/button';
import {ErrorNotice} from '../shared/components/error-notice';
import {UploadButton} from '../shared/components/upload-button';
import {exampleEventSource} from '../shared/examples';
import {EventSource} from '../shared/models';
import * as nsUtils from '../shared/namespaces';
import {services} from '../shared/services';
import {EventSourceEditor} from './event-source-editor';

export function EventSourceCreator({onCreate, namespace}: {namespace: string; onCreate: (eventSource: EventSource) => void}) {
    const [eventSource, setEventSource] = useState<EventSource>(exampleEventSource(nsUtils.getNamespaceWithDefault(namespace)));
    const [error, setError] = useState<Error>();
    return (
        <>
            <div>
                <UploadButton onUpload={setEventSource} onError={setError} />
                <Button
                    icon='plus'
                    onClick={async () => {
                        try {
                            const newEventSource = await services.eventSource.create(eventSource, nsUtils.getNamespaceWithDefault(eventSource.metadata.namespace));
                            onCreate(newEventSource);
                        } catch (err) {
                            setError(err);
                        }
                    }}>
                    Create
                </Button>
            </div>
            <ErrorNotice error={error} />
            <EventSourceEditor eventSource={eventSource} onChange={setEventSource} onError={setError} />
            <p>
                <a href='https://github.com/argoproj/argo-events/tree/stable/examples/event-sources'>
                    Example event sources <i className='fa fa-external-link-alt' />
                </a>
            </p>
        </>
    );
}

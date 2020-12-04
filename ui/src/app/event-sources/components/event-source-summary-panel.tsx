import * as React from 'react';

import {ExampleManifests} from '../../shared/components/example-manifests';
import {ResourceEditor} from '../../shared/components/resource-editor/resource-editor';
import {Timestamp} from '../../shared/components/timestamp';
import {services} from '../../shared/services';
// @ts-ignore
import {EventSource} from '../../../models'

interface Props {
    eventSource: EventSource;
    onChange: (eventSource: EventSource) => void;
}

export const EventSourceSummaryPanel = (props: Props) => {
    const attributes = [
        {title: 'Name', value: props.eventSource.metadata.name},
        {title: 'Created', value: <Timestamp date={props.eventSource.metadata.creationTimestamp} />}
    ];
    return (
        <div>
            <div className='white-box'>
                <div className='white-box__details'>
                    {attributes.map(attr => (
                        <div className='row white-box__details-row' key={attr.title}>
                            <div className='columns small-3'>{attr.title}</div>
                            <div className='columns small-9'>{attr.value}</div>
                        </div>
                    ))}
                </div>
            </div>

            <div className='white-box'>
                <div className='white-box__details'>
                    <ResourceEditor
                        kind='Eventsource'
                        title='Update Event source'
                        value={props.eventSource}
                        onSubmit={(value: EventSource ) =>
                            services.eventSource.update(value, props.eventSource.metadata.namespace).then(eventSource => props.onChange(eventSource))
                        }
                    />
                    <p>
                        <ExampleManifests />
                    </p>
                </div>
            </div>
        </div>
    );
};

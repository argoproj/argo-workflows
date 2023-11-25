import * as React from 'react';

import {Tabs} from 'argo-ui';
import {EventSource} from '../../../models';
import {MetadataEditor} from '../../shared/components/editors/metadata-editor';
import {ObjectEditor} from '../../shared/components/object-editor/object-editor';

export function EventSourceEditor({
    onChange,
    onTabSelected,
    selectedTabKey,
    eventSource
}: {
    eventSource: EventSource;
    onChange: (template: EventSource) => void;
    onError: (error: Error) => void;
    onTabSelected?: (tab: string) => void;
    selectedTabKey?: string;
}) {
    return (
        <Tabs
            key='tabs'
            navTransparent={true}
            selectedTabKey={selectedTabKey}
            onTabSelected={onTabSelected}
            tabs={[
                {
                    key: 'manifest',
                    title: 'Manifest',
                    content: <ObjectEditor type='io.argoproj.events.v1alpha1.EventSource' value={eventSource} onChange={x => onChange({...x})} />
                },

                {
                    key: 'metadata',
                    title: 'MetaData',
                    content: <MetadataEditor value={eventSource.metadata} onChange={metadata => onChange({...eventSource, metadata})} />
                }
            ]}
        />
    );
}

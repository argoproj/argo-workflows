import {Tabs} from 'argo-ui/src/components/tabs/tabs';
import * as React from 'react';

import {MetadataEditor} from '../shared/components/editors/metadata-editor';
import {ObjectEditor} from '../shared/components/object-editor';
import type {Lang} from '../shared/components/object-parser';
import {EventSource} from '../shared/models';

export function EventSourceEditor({
    onChange,
    onLangChange,
    onTabSelected,
    selectedTabKey,
    eventSource,
    serialization,
    lang
}: {
    eventSource: EventSource;
    serialization: string;
    lang: Lang;
    onChange: (template: string | EventSource) => void;
    onLangChange: (lang: Lang) => void;
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
                    content: (
                        <ObjectEditor
                            type='io.argoproj.events.v1alpha1.EventSource'
                            value={eventSource}
                            text={serialization}
                            lang={lang}
                            onLangChange={onLangChange}
                            onChange={onChange}
                        />
                    )
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

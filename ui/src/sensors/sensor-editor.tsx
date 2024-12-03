import {Tabs} from 'argo-ui/src/components/tabs/tabs';
import * as React from 'react';

import {MetadataEditor} from '../shared/components/editors/metadata-editor';
import {ObjectEditor} from '../shared/components/object-editor';
import type {Lang} from '../shared/components/object-parser';
import {Sensor} from '../shared/models';

export function SensorEditor({
    onChange,
    onLangChange,
    onTabSelected,
    selectedTabKey,
    sensor,
    serialization,
    lang
}: {
    sensor: Sensor;
    serialization: string;
    lang: Lang;
    onChange: (template: string | Sensor) => void;
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
                        <ObjectEditor type='io.argoproj.events.v1alpha1.Sensor' value={sensor} text={serialization} lang={lang} onChange={onChange} onLangChange={onLangChange} />
                    )
                },
                {
                    key: 'metadata',
                    title: 'MetaData',
                    content: <MetadataEditor value={sensor.metadata} onChange={metadata => onChange({...sensor, metadata})} />
                }
            ]}
        />
    );
}

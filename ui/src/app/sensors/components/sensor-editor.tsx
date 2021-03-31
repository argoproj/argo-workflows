import * as React from 'react';

import {Tabs} from 'argo-ui';
import {Sensor} from '../../../models';
import {MetadataEditor} from '../../shared/components/editors/metadata-editor';
import {ObjectEditor} from '../../shared/components/object-editor/object-editor';

export const SensorEditor = ({
    onChange,
    onError,
    onTabSelected,
    selectedTabKey,
    sensor
}: {
    sensor: Sensor;
    onChange: (template: Sensor) => void;
    onError: (error: Error) => void;
    onTabSelected?: (tab: string) => void;
    selectedTabKey?: string;
}) => {
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
                    content: <ObjectEditor type='io.argoproj.events.v1alpha1.Sensor' value={sensor} onChange={x => onChange({...x})} />
                },
                {
                    key: 'metadata',
                    title: 'MetaData',
                    content: <MetadataEditor value={sensor.metadata} onChange={metadata => onChange({...sensor, metadata})} />
                }
            ]}
        />
    );
};

import {SlidingPanel} from 'argo-ui/src/components/sliding-panel/sliding-panel';
import {Tabs} from 'argo-ui/src/components/tabs/tabs';
import * as React from 'react';
import {useState} from 'react';

import {Node} from '../shared/components/graph/types';
import {Sensor} from '../shared/models';
import {EventsPanel} from '../workflows/components/events-panel';
import {SensorLogsViewer} from './sensor-logs-viewer';

export function SensorSidePanel({
    isShown,
    namespace,
    sensor,
    selectedTrigger,
    onTriggerClicked,
    onClose
}: {
    isShown: boolean;
    namespace: string;
    sensor: Sensor;
    selectedTrigger: string;
    onTriggerClicked: (selectedNode: string) => void;
    onClose: () => void;
}) {
    const queryParams = new URLSearchParams(location.search);
    const [logTab, setLogTab] = useState<Node>(queryParams.get('logTab'));

    return (
        <SlidingPanel isShown={isShown} onClose={onClose}>
            {!!sensor && (
                <div>
                    <h4>
                        {sensor.metadata.name}
                        {selectedTrigger ? '/' + selectedTrigger : ''}
                    </h4>
                    <Tabs
                        navTransparent={true}
                        selectedTabKey={logTab}
                        onTabSelected={setLogTab}
                        tabs={[
                            {
                                title: 'LOGS',
                                key: 'logs',
                                content: <SensorLogsViewer namespace={namespace} selectedTrigger={selectedTrigger} sensor={sensor} onClick={onTriggerClicked} />
                            },
                            {
                                title: 'EVENTS',
                                key: 'events',
                                content: <EventsPanel kind='Sensor' namespace={namespace} name={sensor.metadata.name} />
                            }
                        ]}
                    />
                </div>
            )}
        </SlidingPanel>
    );
}

import {useState} from 'react';

import {ScopedLocalStorage} from './scoped-local-storage';

export enum TIMESTAMP_KEYS {
    WORKFLOW_NODE_STARTED = 'workflowNodeStarted',
    WORKFLOW_NODE_FINISHED = 'workflowNodeFinished',
    CLUSTER_WORKFLOW_TEMPLATE_LIST = 'clusterWorkflowTemplateList',
    CRON_WORKFLOW_LIST_CREATION = 'cronWorkflowListCreation',
    CRON_WORKFLOW_LIST_NEXT_SCHEDULED = 'cronWorkflowListNextScheduled',
    CRON_WORKFLOW_STATUS_LAST_SCHEDULED = 'cronWorkflowStatusLastScheduled',
    EVENT_SOURCE_LIST_CREATION = 'eventSourceListCreation',
    SENSOR_LIST_CREATION = 'sensorListCreation',
    EVENTS_PANEL_LAST = 'eventsPanelLast',
    WORKFLOW_ARTIFACTS_CREATED = 'workflowArtifactsCreated',
    WORKFLOW_SUMMARY_PANEL_START = 'workflowSummaryPanelStart',
    WORKFLOW_SUMMARY_PANEL_END = 'workflowSummaryPanelEnd',
    WORKFLOW_NODE_ARTIFACT_CREATED = 'workflowNodeArtifactCreated',
    WORKFLOW_TEMPLATE_LIST_CREATION = 'workflowTemplateListCreation',
    WORKFLOWS_ROW_STARTED = 'workflowsRowStarted',
    WORKFLOWS_ROW_FINISHED = 'workflowsRowFinished',
    CRON_ROW_STARTED = 'cronRowStarted',
    CRON_ROW_FINISHED = 'cronRowFinished'
}

const storage = new ScopedLocalStorage('Timestamp');

// key is used to store the preference in local storage
const useTimestamp = (timestampKey: TIMESTAMP_KEYS): [boolean, (value: boolean) => void] => {
    const [storedDisplayISOFormat, setStoredDisplayISOFormat] = useState<boolean>(storage.getItem(`displayISOFormat-${timestampKey}`, false));

    const handleStoredDisplayISOFormatChange = (value: boolean) => {
        setStoredDisplayISOFormat(value);
        storage.setItem(`displayISOFormat-${timestampKey}`, value, false);
    };

    return [storedDisplayISOFormat, handleStoredDisplayISOFormatChange];
};

export default useTimestamp;

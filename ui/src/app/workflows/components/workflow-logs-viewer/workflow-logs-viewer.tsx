import * as React from 'react';
import {useEffect, useState} from 'react';

import {Autocomplete} from 'argo-ui';
import {Observable} from 'rxjs';
import * as models from '../../../../models';
import {execSpec} from '../../../../models';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {InfoIcon, WarningIcon} from '../../../shared/components/fa-icons';
import {Links} from '../../../shared/components/links';
import {services} from '../../../shared/services';
import {FullHeightLogsViewer} from './full-height-logs-viewer';

interface WorkflowLogsViewerProps {
    workflow: models.Workflow;
    nodeId?: string;
    container: string;
    archived: boolean;
}

function identity<T>(value: T) {
    return () => value;
}

export const WorkflowLogsViewer = ({workflow, nodeId, container, archived}: WorkflowLogsViewerProps) => {
    const [podName, setPodName] = useState(nodeId);
    const [selectedContainer, setContainer] = useState(container);
    const [error, setError] = useState<Error>();
    const [loaded, setLoaded] = useState(false);
    const [logsObservable, setLogsObservable] = useState<Observable<string>>();

    useEffect(() => {
        setError(null);
        setLoaded(false);
        const source = services.workflows
            .getContainerLogs(workflow, podName, selectedContainer, archived)
            .map(e => (!podName ? e.podName + ': ' : '') + e.content + '\n')
            .publishReplay()
            .refCount();
        const subscription = source.subscribe(() => setLoaded(true), setError);
        setLogsObservable(source);
        return () => subscription.unsubscribe();
    }, [workflow.metadata.namespace, workflow.metadata.name, podName, selectedContainer, archived]);

    const podNameOptions = [{value: null, label: 'All'}].concat(
        Object.values(workflow.status.nodes || {})
            .filter(x => x.type === 'Pod')
            .map(x => ({value: x.id, label: (x.displayName || x.name) + ' (' + x.id + ')'}))
    );

    const containers = ['main', 'init', 'wait'];
    return (
        <div className='workflow-logs-viewer'>
            <h3>Logs</h3>
            {archived && (
                <p>
                    <i className='fa fa-exclamation-triangle' /> Logs for archived workflows may be overwritten by a more recent workflow with the same name.
                </p>
            )}
            <p>
                <i className='fa fa-box' />{' '}
                <Autocomplete items={podNameOptions} value={(podNameOptions.find(x => x.value === podName) || {}).label} onSelect={(_, item) => setPodName(item.value)} /> /{' '}
                <Autocomplete items={containers} value={selectedContainer} onSelect={setContainer} />
            </p>
            <ErrorNotice error={error} />
            {selectedContainer === 'init' && (
                <p>
                    <InfoIcon /> Init containers logs are usually only useful when debugging input artifact problems. The init container is only run if there were input artifacts.
                </p>
            )}
            {selectedContainer === 'wait' && (
                <p>
                    <InfoIcon /> Wait containers logs are usually only useful when debugging output artifact problems. The wait container is only run if there were output artifacts
                    (including archived logs).
                </p>
            )}
            {!loaded ? (
                <p className='white-box'>
                    <i className='fa fa-circle-notch fa-spin' /> Waiting for data...
                </p>
            ) : (
                <FullHeightLogsViewer
                    source={{
                        key: `${workflow.metadata.name}-${podName}-${selectedContainer}`,
                        loadLogs: identity(logsObservable),
                        shouldRepeat: () => false
                    }}
                />
            )}
            <p>
                {podName && (
                    <>
                        Still waiting for data or an error? Try getting{' '}
                        <a href={services.workflows.getArtifactLogsUrl(workflow, podName, selectedContainer, archived)}>logs from the artifacts</a>.
                    </>
                )}
                {execSpec(workflow).podGC && (
                    <>
                        <WarningIcon /> You pod GC settings will delete pods and their logs immediately on completion.
                    </>
                )}{' '}
                Logs do not appear for pods that are deleted.{' '}
                {podName ? (
                    <Links
                        object={{
                            metadata: {
                                namespace: workflow.metadata.namespace,
                                name: podName
                            },
                            status: {
                                startedAt: workflow.status.startedAt,
                                finishedAt: workflow.status.finishedAt
                            }
                        }}
                        scope='pod-logs'
                    />
                ) : (
                    <Links object={workflow} scope='workflow' />
                )}
            </p>
        </div>
    );
};

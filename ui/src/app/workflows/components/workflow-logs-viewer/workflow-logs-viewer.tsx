import * as React from 'react';
import {useEffect, useState} from 'react';

import {Observable} from 'rxjs';
import * as models from '../../../../models';
import {execSpec} from '../../../../models';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {InfoIcon, WarningIcon} from '../../../shared/components/fa-icons';
import {services} from '../../../shared/services';
import {FullHeightLogsViewer} from './full-height-logs-viewer';

require('./workflow-logs-viewer.scss');

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

    const podNameOptions = [{value: '', title: 'All'}].concat(
        Object.values(workflow.status.nodes || {})
            .filter(x => x.type === 'Pod')
            .map(x => ({value: x.id, title: x.displayName || x.name}))
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
                <select className='select' value={podName} onChange={x => setPodName(podNameOptions[x.target.selectedIndex].value)}>
                    {podNameOptions.map(x => (
                        <option key={x.value} value={x.value}>
                            {x.title}
                        </option>
                    ))}
                </select>{' '}
                /{' '}
                <select className='select' value={selectedContainer} onChange={x => setContainer(containers[x.target.selectedIndex])}>
                    {containers.map(x => (
                        <option key={x} value={x}>
                            {x}
                        </option>
                    ))}
                </select>
            </p>
            {error && <ErrorNotice error={error} />}
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
            <div className='white-box'>
                {!loaded ? (
                    <p>
                        <i className='fa fa-circle-notch fa-spin' /> Waiting for data...
                    </p>
                ) : (
                    <div className='log-box'>
                        <FullHeightLogsViewer
                            source={{
                                key: `${workflow.metadata.name}-${podName}-${selectedContainer}`,
                                loadLogs: identity(logsObservable),
                                shouldRepeat: () => false
                            }}
                        />
                    </div>
                )}
            </div>
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
                Logs do not appear for pods that are deleted.
            </p>
        </div>
    );
};

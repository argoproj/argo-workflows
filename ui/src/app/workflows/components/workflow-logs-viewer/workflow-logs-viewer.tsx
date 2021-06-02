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
    const [podName, setPodName] = useState(nodeId || '');
    const [selectedContainer, setContainer] = useState(container);
    const [grep, setGrep] = useState('');
    const [error, setError] = useState<Error>();
    const [loaded, setLoaded] = useState(false);
    const [logsObservable, setLogsObservable] = useState<Observable<string>>();

    useEffect(() => {
        setError(null);
        setLoaded(false);
        const source = services.workflows
            .getContainerLogs(workflow, podName, selectedContainer, grep, archived)
            .map(e => (!podName ? e.podName + ': ' : '') + e.content + '\n')
            // this next line highlights the search term in bold with a yellow background, white text
            .map(x => x.replace(new RegExp(grep, 'g'), y => '\u001b[1m\u001b[43;1m\u001b[37m' + y + '\u001b[0m'))
            .publishReplay()
            .refCount();
        const subscription = source.subscribe(
            () => setLoaded(true),
            setError,
            () => setLoaded(true)
        );
        setLogsObservable(source);
        return () => subscription.unsubscribe();
    }, [workflow.metadata.namespace, workflow.metadata.name, podName, selectedContainer, grep, archived]);

    // filter allows us to introduce a short delay, before we actually change grep
    const [filter, setFilter] = useState('');
    useEffect(() => {
        const x = setTimeout(() => setGrep(filter), 1000);
        return () => clearTimeout(x);
    }, [filter]);

    const podNames = [{value: '', label: 'All'}].concat(
        Object.values(workflow.status.nodes || {})
            .filter(x => x.type === 'Pod')
            .map(x => ({value: x.id, label: (x.displayName || x.name) + ' (' + x.id + ')'}))
    );

    const node = workflow.status.nodes[nodeId];
    const templates = execSpec(workflow).templates.filter(t => !node || t.name === node.templateName);

    const containers = ['init', 'wait'].concat(
        templates
            .map(t => ((t.containerSet && t.containerSet.containers) || [{name: 'main'}]).concat(t.sidecars || []))
            .reduce((a, v) => a.concat(v), [])
            .map(c => c.name)
    );

    return (
        <div className='workflow-logs-viewer'>
            <h3>Logs</h3>
            {archived && (
                <p>
                    <i className='fa fa-exclamation-triangle' /> Logs for archived workflows may be overwritten by a more recent workflow with the same name.
                </p>
            )}
            <div>
                <i className='fa fa-box' />{' '}
                <Autocomplete items={podNames} value={(podNames.find(x => x.value === podName) || {label: ''}).label} onSelect={(_, item) => setPodName(item.value)} /> /{' '}
                <Autocomplete items={containers} value={selectedContainer} onSelect={setContainer} />
                <span className='fa-pull-right'>
                    <i className='fa fa-filter' /> <input type='search' defaultValue={filter} onChange={v => setFilter(v.target.value)} placeholder='Filter (regexp)...' />
                </span>
            </div>
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
            </p>
            <p>
                {execSpec(workflow).podGC && (
                    <>
                        <WarningIcon /> Your pod GC settings will delete pods and their logs immediately on completion.
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
                            workflow,
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

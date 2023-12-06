import * as React from 'react';
import {useContext, useEffect, useState} from 'react';

import {Autocomplete} from 'argo-ui';
import {Observable} from 'rxjs';
import {map, publishReplay, refCount} from 'rxjs/operators';
import * as models from '../../../../models';
import {execSpec} from '../../../../models';
import {ANNOTATION_KEY_POD_NAME_VERSION} from '../../../shared/annotations';
import {Button} from '../../../shared/components/button';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {InfoIcon, WarningIcon} from '../../../shared/components/fa-icons';
import {Links} from '../../../shared/components/links';
import {Context} from '../../../shared/context';
import {useLocalStorage} from '../../../shared/hooks/uselocalstorage';
import {getPodName, getTemplateNameFromNode} from '../../../shared/pod-name';
import {ScopedLocalStorage} from '../../../shared/scoped-local-storage';
import {services} from '../../../shared/services';
import {FullHeightLogsViewer} from './full-height-logs-viewer';
import {extractJsonValue, JsonLogsFieldSelector, SelectedJsonFields} from './json-logs-field-selector';

const TZ_LOCALSTORAGE_KEY = 'DEFAULT_TZ';
const DEFAULT_TZ = process.env.DEFAULT_TZ || 'UTC';
const timezones = Intl.supportedValuesOf('timeZone');

interface WorkflowLogsViewerProps {
    workflow: models.Workflow;
    nodeId?: string;
    initialPodName: string;
    container: string;
    archived: boolean;
}

function identity<T>(value: T) {
    return () => value;
}

// USED FOR MANUAL TESTING
// const timeSpammer:Observable<string> = new Observable((subscriber) => {
//   setInterval(() => {
//     subscriber.next('time="2022-11-27T04:07:37.291Z" level=info msg="running spammer" argo=true\n');
//   }, 2000);
// });

interface ParsedTime {
    quoted: string;
    fullstring: string;
}
// extract the time field from a string
function parseTime(formattedString: string): undefined | ParsedTime {
    const re = new RegExp('time="(.*?)"');
    const table = re.exec(formattedString);
    if (table === null || table.length !== 2) {
        return undefined;
    }
    return {quoted: table[1], fullstring: table[0]};
}

function parseAndTransform(formattedString: string, timeZone: string) {
    const maybeTime = parseTime(formattedString);
    if (maybeTime === undefined) {
        return formattedString;
    }

    try {
        // hack to get a local ISO time: en-CA locale is very close to ISO (https://en.wikipedia.org/wiki/Date_and_time_notation_in_Canada)
        const newTime = new Date(maybeTime.quoted).toLocaleString('en-CA', {timeZone, hour12: false}).replace(', ', 'T');
        const shortTz = new Date().toLocaleTimeString('en-US', {timeZone, timeZoneName: 'short'}).split(' ')[2];
        const newFormattedTime = `time="${newTime} ${shortTz}"`;
        const newFormattedString = formattedString.replace(maybeTime.fullstring, newFormattedTime);
        return newFormattedString;
    } catch {
        return formattedString;
    }
}

export function WorkflowLogsViewer({workflow, nodeId, initialPodName, container, archived}: WorkflowLogsViewerProps) {
    const storage = new ScopedLocalStorage('workflow-logs-viewer');
    const storedJsonFields = storage.getItem('jsonFields', {
        values: []
    } as SelectedJsonFields);

    const {popup} = useContext(Context);
    const [podName, setPodName] = useState(initialPodName || '');
    const [selectedContainer, setContainer] = useState(container);
    const [grep, setGrep] = useState('');
    const [error, setError] = useState<Error>();
    const [loaded, setLoaded] = useState(false);
    const [logsObservable, setLogsObservable] = useState<Observable<string>>();
    // timezone used for ui rendering only
    const [uiTimezone, setUITimezone] = useState<string>(DEFAULT_TZ);
    // timezone used for timezone formatting
    const [timezone, setTimezone] = useLocalStorage<string>(TZ_LOCALSTORAGE_KEY, DEFAULT_TZ);

    // update the UI everytime the timezone changes
    useEffect(() => {
        setUITimezone(timezone);
    }, [timezone]);
    const [selectedJsonFields, setSelectedJsonFields] = useState<SelectedJsonFields>(storedJsonFields);

    useEffect(() => {
        setError(null);
        setLoaded(false);
        const source = services.workflows.getContainerLogs(workflow, podName, nodeId, selectedContainer, grep, archived).pipe(
            // extract message from LogEntry
            map(e => {
                const values: string[] = [];
                const content = e.content;
                if (selectedJsonFields.values.length > 0) {
                    try {
                        const json = JSON.parse(content);
                        selectedJsonFields.values.forEach(selectedJsonField => {
                            const value = extractJsonValue(json, selectedJsonField);
                            if (value) {
                                values.push(value);
                            }
                        });
                    } catch (e) {
                        // if not json, show content directly
                    }
                }
                if (values.length === 0) {
                    values.push(content);
                }
                return `${!podName ? e.podName + ': ' : ''}${values.join(' ')}\n`;
            }),
            // this next line highlights the search term in bold with a yellow background, white text
            map(x => {
                if (grep !== '') {
                    return x.replace(new RegExp(grep, 'g'), y => '\u001b[1m\u001b[43;1m\u001b[37m' + y + '\u001b[0m');
                }
                return x;
            }),
            map((x: string) => parseAndTransform(x, timezone)),
            publishReplay(),
            refCount()
        );

        // const source = timeSpammer.pipe(
        //   map((x)=> parseAndTransform(x, timezone)),
        //   publishReplay(),
        //   refCount()
        // );
        const subscription = source.subscribe(
            () => setLoaded(true),
            setError,
            () => setLoaded(true)
        );
        setLogsObservable(source);
        return () => subscription.unsubscribe();
    }, [workflow.metadata.namespace, workflow.metadata.name, podName, selectedContainer, grep, archived, selectedJsonFields, timezone]);

    // filter allows us to introduce a short delay, before we actually change grep
    const [logFilter, setLogFilter] = useState('');
    useEffect(() => {
        const x = setTimeout(() => setGrep(logFilter), 1000);
        return () => clearTimeout(x);
    }, [logFilter]);

    const annotations = workflow.metadata.annotations || {};
    const podNameVersion = annotations[ANNOTATION_KEY_POD_NAME_VERSION];

    // map pod names to corresponding node IDs
    const podNamesToNodeIDs = new Map<string, string>();

    const podNames = [{value: '', label: 'All'}].concat(
        Object.values(workflow.status.nodes || {})
            .filter(x => x.type === 'Pod')
            .map(targetNode => {
                const {name, id, displayName} = targetNode;
                const templateName = getTemplateNameFromNode(targetNode);
                const targetPodName = getPodName(workflow.metadata.name, name, templateName, id, podNameVersion);
                podNamesToNodeIDs.set(targetPodName, id);
                return {value: targetPodName, label: (displayName || name) + ' (' + targetPodName + ')'};
            })
    );

    const node = workflow.status.nodes[nodeId];
    const templates = execSpec(workflow).templates.filter(t => !node || t.name === node.templateName);

    const containers = [
        ...new Set(
            ['init', 'wait'].concat(
                templates
                    .map(t => ((t.containerSet && t.containerSet.containers) || [{name: 'main'}]).concat(t.sidecars || []).concat(t.initContainers || []))
                    .reduce((a, v) => a.concat(v), [])
                    .map(c => c.name)
            )
        )
    ];
    const [candidateContainer, setCandidateContainer] = useState(container);
    const filteredTimezones = timezones.filter(tz => tz.startsWith(uiTimezone) || uiTimezone === '');

    async function popupJsonFieldSelector() {
        const fields = {...selectedJsonFields};
        const updated = await popup.confirm('Select Json Fields', () => (
            <JsonLogsFieldSelector
                fields={selectedJsonFields}
                onChange={values => {
                    fields.values = values;
                }}
            />
        ));
        if (updated) {
            storage.setItem('jsonFields', fields, {values: []});
            setSelectedJsonFields(fields);
        }
    }

    return (
        <div className='workflow-logs-viewer'>
            <h3>Logs</h3>
            {archived && (
                <p>
                    <i className='fa fa-exclamation-triangle' /> Logs for archived workflows may be overwritten by a more recent workflow with the same name.
                </p>
            )}
            <div style={{marginBottom: 10}}>
                <i className='fa fa-box' />{' '}
                <Autocomplete
                    items={podNames}
                    value={(podNames.find(x => x.value === podName) || {label: ''}).label}
                    onSelect={(_, item) => {
                        setPodName(item.value);
                    }}
                />{' '}
                /{' '}
                <Autocomplete
                    items={containers}
                    value={candidateContainer}
                    onSelect={v => {
                        setCandidateContainer(v);
                        setContainer(v);
                    }}
                    onChange={v => setCandidateContainer(v.target.value)}
                    renderInput={props => (
                        <input
                            {...props}
                            onKeyUp={event => {
                                if (event.keyCode === 13) {
                                    // ENTER, to confirm custom container name input
                                    setContainer(candidateContainer);
                                }
                            }}
                        />
                    )}
                />
                <Button onClick={popupJsonFieldSelector} icon={'exchange-alt'}>
                    Log Fields
                </Button>
                <span className='fa-pull-right'>
                    <div className='log-menu'>
                        <i className='fa fa-filter' />{' '}
                        <input type='search' defaultValue={logFilter} onChange={v => setLogFilter(v.target.value)} placeholder='Filter (regexp)...' />
                        <i className='fa fa-globe' />{' '}
                        <Autocomplete
                            items={filteredTimezones}
                            value={uiTimezone}
                            onChange={v => setUITimezone(v.target.value)}
                            // useEffect ensures UITimezone is also changed
                            onSelect={setTimezone}
                        />
                    </div>
                </span>
            </div>
            <ErrorNotice error={error} />
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
                {selectedContainer === 'init' && (
                    <>
                        <InfoIcon /> Init containers logs are useful when debugging input artifact problems.
                    </>
                )}
                {selectedContainer === 'wait' && (
                    <>
                        {' '}
                        <InfoIcon /> Wait containers logs are useful when debugging output artifact problems.
                    </>
                )}
                {podName && podNamesToNodeIDs.get(podName) && (
                    <>
                        Still waiting for data or an error? Try getting{' '}
                        <a href={services.workflows.getArtifactLogsPath(workflow, podNamesToNodeIDs.get(podName), selectedContainer, archived)}>logs from the artifacts</a>.
                    </>
                )}
                {execSpec(workflow).podGC && (
                    <>
                        <WarningIcon /> Your pod GC settings will delete pods and their logs immediately on completion.
                    </>
                )}{' '}
                Logs may not appear for pods that are deleted.{' '}
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
}

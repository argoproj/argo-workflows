import {Tabs, Ticker, Tooltip} from 'argo-ui';
import moment from 'moment';
import * as React from 'react';
import {useState} from 'react';

import * as models from '../../../../models';
import {Artifact, NodeStatus, Workflow} from '../../../../models';
import {ANNOTATION_KEY_POD_NAME_VERSION} from '../../../shared/annotations';
import {Button} from '../../../shared/components/button';
import {ClipboardText} from '../../../shared/components/clipboard-text';
import {DurationPanel} from '../../../shared/components/duration-panel';
import {InlineTable} from '../../../shared/components/inline-table/inline-table';
import {Links} from '../../../shared/components/links';
import {Phase} from '../../../shared/components/phase';
import {Timestamp} from '../../../shared/components/timestamp';
import {getPodName, getTemplateNameFromNode} from '../../../shared/pod-name';
import {ResourcesDuration} from '../../../shared/resources-duration';
import {services} from '../../../shared/services';
import {getResolvedTemplates} from '../../../shared/template-resolution';

import './workflow-node-info.scss';

function nodeDuration(node: models.NodeStatus, now: moment.Moment) {
    const endTime = node.finishedAt ? moment(node.finishedAt) : now;
    return endTime.diff(moment(node.startedAt)) / 1000;
}

// Iterate over the node's subtree and find pod in error or fail
function failHosts(node: models.NodeStatus, workflow: models.Workflow) {
    const hosts = new Array<string>();
    const toVisit = [node.id];
    while (toVisit.length > 0) {
        const nodeNameToVisit = toVisit[toVisit.length - 1];
        toVisit.pop();

        if (nodeNameToVisit in workflow.status.nodes) {
            const nodeToVisit = workflow.status.nodes[nodeNameToVisit];
            if (
                nodeToVisit.type === 'Pod' &&
                (nodeToVisit.phase === models.NODE_PHASE.FAILED || nodeToVisit.phase === models.NODE_PHASE.ERROR) &&
                hosts.indexOf(nodeToVisit.hostNodeName) === -1
            ) {
                hosts.push(nodeToVisit.hostNodeName);
            }
            if (nodeToVisit.children) {
                for (const child of nodeToVisit.children) {
                    toVisit.push(child);
                }
            }
        }
    }
    const uniqueHosts = hosts.filter((v: string, i: number) => hosts.indexOf(v) === i);
    return uniqueHosts.join('\n');
}

interface Props {
    node: models.NodeStatus;
    workflow: models.Workflow;
    links: models.Link[];
    archived: boolean;
    onShowContainerLogs: (nodeId: string, container: string) => any;
    onShowEvents?: () => void;
    onShowYaml?: (nodeId: string) => any;
    onTabSelected?: (tabSelected: string) => void;
    selectedTabKey?: string;
    onResume?: () => void;
}

const AttributeRow = (attr: {title: string; value: any}) => (
    <React.Fragment key={attr.title}>
        <div>{attr.title}</div>
        <div>{attr.value}</div>
    </React.Fragment>
);
const AttributeRows = (props: {attributes: {title: string; value: any}[]}) => (
    <div className='workflow-details__attribute-grid'>
        {props.attributes.map(attr => (
            <AttributeRow key={attr.title} {...attr} />
        ))}
    </div>
);

function DisplayWorkflowTime(props: {date: Date | string | number}) {
    const {date} = props;
    const getLocalDateTime = (utc: Date | string | number) => {
        return new Date(utc.toString()).toLocaleString();
    };
    return (
        <div>
            {date === null || date === undefined ? (
                '-'
            ) : (
                <span>
                    {getLocalDateTime(date)} (<Timestamp date={date} />)
                </span>
            )}
        </div>
    );
}

function WorkflowNodeSummary(props: Props) {
    const {workflow, node} = props;

    const annotations = workflow.metadata.annotations || {};
    const version = annotations[ANNOTATION_KEY_POD_NAME_VERSION];
    const templateName = getTemplateNameFromNode(node);

    const podName = getPodName(workflow.metadata.name, node.name, templateName, node.id, version);

    const attributes = [
        {title: 'NAME', value: <ClipboardText text={props.node.name} />},
        {title: 'ID', value: <ClipboardText text={props.node.id} />},
        {title: 'TYPE', value: props.node.type},
        {
            title: 'PHASE',
            value: <Phase value={props.node.phase} />
        },
        ...(props.node.message
            ? [
                  {
                      title: 'MESSAGE',
                      value: <span className='workflow-node-info__multi-line'>{props.node.message}</span>
                  }
              ]
            : []),
        {title: 'START TIME', value: <DisplayWorkflowTime date={props.node.startedAt} />},
        {title: 'END TIME', value: <DisplayWorkflowTime date={props.node.finishedAt} />},
        {
            title: 'DURATION',
            value: <Ticker>{now => <DurationPanel duration={nodeDuration(props.node, now)} phase={props.node.phase} estimatedDuration={props.node.estimatedDuration} />}</Ticker>
        },
        {title: 'PROGRESS', value: props.node.progress || '-'},
        {
            title: 'MEMOIZATION',
            value: (
                <InlineTable
                    rows={
                        props.node.memoizationStatus
                            ? [
                                  {
                                      left: <div> KEY </div>,
                                      right: <div> {props.node.memoizationStatus.key} </div>
                                  },
                                  {
                                      left: <div> CACHE NAME </div>,
                                      right: <div> {props.node.memoizationStatus.cacheName} </div>
                                  },
                                  {
                                      left: <div> HIT? </div>,
                                      right: <div> {props.node.memoizationStatus.hit ? 'YES' : 'NO'} </div>
                                  }
                              ]
                            : [{left: <div> N/A </div>, right: null}]
                    }
                />
            )
        }
    ];
    if (props.node.type === 'Pod') {
        attributes.splice(
            2,
            0,
            {title: 'POD NAME', value: <ClipboardText text={podName} />},
            {
                title: 'HOST NODE NAME',
                value: <ClipboardText text={props.node.hostNodeName} />
            }
        );
    }
    if (props.node.type === 'Retry') {
        attributes.push({
            title: 'FAIL HOSTS',
            value: <pre className='workflow-node-info__multi-line'>{failHosts(props.node, props.workflow)}</pre>
        });
    }
    if (props.node.resourcesDuration) {
        attributes.push({
            title: 'RESOURCES DURATION',
            value: <ResourcesDuration resourcesDuration={props.node.resourcesDuration} />
        });
    }
    const showLogs = (x = 'main') => props.onShowContainerLogs(props.node.id, x);
    return (
        <div className='white-box'>
            <div className='white-box__details' style={{paddingBottom: '8px'}}>
                {<AttributeRows attributes={attributes} />}
            </div>
            <div>
                {props.node.type === 'Suspend' && props.onResume && (
                    <Button icon='play' onClick={() => props.onResume()}>
                        RESUME
                    </Button>
                )}{' '}
                {props.node.type !== 'Container' && props.onShowYaml && (
                    <Button icon='file-code' onClick={() => props.onShowYaml(props.node.id)}>
                        MANIFEST
                    </Button>
                )}{' '}
                {props.node.type === 'Pod' && props.onShowContainerLogs && (
                    <Button onClick={() => showLogs()} icon='bars'>
                        LOGS
                    </Button>
                )}{' '}
                {props.node.type === 'Pod' && props.onShowEvents && (
                    <Button icon='bell' onClick={() => props.onShowEvents()}>
                        EVENTS
                    </Button>
                )}{' '}
                {props.node.type === 'Container' && props.onShowContainerLogs && (
                    <Button
                        icon='bars'
                        onClick={() =>
                            props.onShowContainerLogs(
                                // find parent node id using node name,
                                // in container set, the parent of the selected node id contains log output
                                Object.keys(props.workflow.status.nodes).find(key => props.workflow.status.nodes[key].name === props.node.name.replace(/.[^.]*$/, '')),
                                props.node.name.replace(/.*\./, '')
                            )
                        }>
                        logs
                    </Button>
                )}{' '}
                {props.node.type === 'Pod' && (
                    <Links
                        button={true}
                        object={{
                            metadata: {
                                namespace: props.workflow.metadata.namespace,
                                name: podName
                            },
                            workflow: props.workflow,
                            status: {
                                startedAt: props.node.startedAt,
                                finishedAt: props.node.finishedAt
                            }
                        }}
                        scope='pod'
                    />
                )}
            </div>
        </div>
    );
}

const WorkflowNodeInputs = (props: Props) => (
    <>
        <h5>Inputs</h5>
        <WorkflowNodeParameters parameters={props.node.inputs && props.node.inputs.parameters} />
        <WorkflowNodeArtifacts {...props} isInput={true} artifacts={props.node.inputs && props.node.inputs.artifacts} />
    </>
);

const WorkflowNodeOutputs = (props: Props) => (
    <>
        <h5>Outputs</h5>
        <div className='white-box'>
            <div className='white-box__details'>
                <div className='row'>
                    <WorkflowNodeResult result={props.node.outputs && props.node.outputs.result} />
                    <WorkflowNodeExitCode exitCode={props.node.outputs && props.node.outputs.exitCode} />
                </div>
            </div>
        </div>
        <WorkflowNodeParameters parameters={props.node.outputs && props.node.outputs.parameters} />
        <WorkflowNodeArtifacts {...props} isInput={false} artifacts={props.node.outputs && props.node.outputs.artifacts} />
    </>
);

function WorkflowNodeParameters({parameters}: {parameters: models.Parameter[]}) {
    return (
        <div className='white-box'>
            <div className='white-box__details'>
                {parameters && parameters.length > 0 ? (
                    <>
                        <div className='row white-box__details-row' key='title'>
                            <p>Parameters</p>
                        </div>
                        <AttributeRows key='attrs' attributes={parameters.map(x => ({title: x.name, value: x.value}))} />
                    </>
                ) : (
                    <div className='row'>
                        <div className='columns small-12 text-center'>No parameters</div>
                    </div>
                )}
            </div>
        </div>
    );
}

const WorkflowNodeResult = ({result}: {result: string}) =>
    result ? (
        <>
            <div className='columns small-3'>Result</div>
            <div className='columns small-3'>{result}</div>
        </>
    ) : (
        <div className='columns small-6 text-center'>No result</div>
    );

const WorkflowNodeExitCode = ({exitCode}: {exitCode: number}) =>
    exitCode ? (
        <>
            <div className='columns small-3'>Exit code</div>
            <div className='columns small-3'>{exitCode}</div>
        </>
    ) : (
        <div className='columns 6 text-center'>No exit code</div>
    );

function hasEnv(container: models.kubernetes.Container | models.UserContainer | models.Script): container is models.kubernetes.Container | models.UserContainer {
    return (container as models.kubernetes.Container | models.UserContainer).env !== undefined;
}

function EnvVar(props: {env: models.kubernetes.EnvVar}) {
    const {env} = props;
    const secret = env.valueFrom?.secretKeyRef;
    const secretValue = secret ? (
        <>
            <Tooltip content={'The value of this environment variable has been hidden for security reasons because it comes from a kubernetes secret.'} arrow={false}>
                <i className='fa fa-key' />
            </Tooltip>
            {secret.name}/{secret.key}
        </>
    ) : (
        undefined
    );

    return (
        <pre key={env.name}>
            {env.name}={env.value || secretValue}
        </pre>
    );
}

function WorkflowNodeContainer(props: {
    nodeId: string;
    container: models.kubernetes.Container | models.UserContainer | models.Script;
    onShowContainerLogs: (nodeId: string, container: string) => any;
    onShowEvents: () => void;
}) {
    const container = {name: 'main', args: Array<string>(), source: '', ...props.container};
    const maybeQuote = (v: string) => (v.includes(' ') ? `'${v}'` : v);
    const attributes = [
        {title: 'NAME', value: container.name || 'main'},
        {title: 'IMAGE', value: container.image},
        {
            title: 'COMMAND',
            value: <pre className='workflow-node-info__multi-line'>{(container.command || []).map(maybeQuote).join(' ')}</pre>
        },
        container.source
            ? {title: 'SOURCE', value: <pre className='workflow-node-info__multi-line'>{container.source}</pre>}
            : {
                  title: 'ARGS',
                  value: <pre className='workflow-node-info__multi-line'>{(container.args || []).map(maybeQuote).join(' ')}</pre>
              },
        hasEnv(container)
            ? {
                  title: 'ENV',
                  value: (
                      <pre className='workflow-node-info__multi-line'>
                          {(container.env || []).map(e => (
                              <EnvVar key={e.name} env={e} />
                          ))}
                      </pre>
                  )
              }
            : {title: 'ENV', value: <pre className='workflow-node-info__multi-line' />}
    ];
    return (
        <div className='white-box'>
            <div className='white-box__details'>{<AttributeRows attributes={attributes} />}</div>
            <div>
                <Button outline={true} icon='bars' onClick={() => props.onShowContainerLogs && props.onShowContainerLogs(props.nodeId, container.name)}>
                    LOGS
                </Button>
            </div>
        </div>
    );
}

function WorkflowNodeContainers(props: Props) {
    const [selectedSidecar, setSelectedSidecar] = useState(null);

    const template = getResolvedTemplates(props.workflow, props.node);
    if (!template || (!template.container && !template.containerSet && !template.script)) {
        return (
            <div className='white-box'>
                <div className='row'>
                    <div className='columns small-12 text-center'>No data to display</div>
                </div>
            </div>
        );
    }

    const containers = (template.containerSet ? template.containerSet.containers : []).concat(template.sidecars || []);
    const container = (selectedSidecar && containers.find(item => item.name === selectedSidecar)) || template.container || template.script;

    return (
        <div className='workflow-node-info__containers'>
            {selectedSidecar && <i className='fa fa-angle-left workflow-node-info__sidecar-back' onClick={() => setSelectedSidecar(null)} />}
            <WorkflowNodeContainer nodeId={props.node.id} container={container} onShowContainerLogs={props.onShowContainerLogs} onShowEvents={props.onShowEvents} />
            {!selectedSidecar && (
                <div>
                    <p>SIDECARS:</p>
                    {containers.map(sidecar => (
                        <div className='workflow-node-info__sidecar' key={sidecar.name} onClick={() => setSelectedSidecar(sidecar.name)}>
                            <span>{sidecar.name}</span> <i className='fa fa-angle-right' />
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}

function WorkflowNodeArtifacts(props: {workflow: Workflow; node: NodeStatus; archived: boolean; isInput: boolean; artifacts: Artifact[]}) {
    const artifacts =
        (props.artifacts &&
            props.artifacts.map(artifact =>
                Object.assign({}, artifact, {
                    downloadUrl: services.workflows.getArtifactDownloadUrl(props.workflow, props.node.id, artifact.name, props.archived, props.isInput),
                    stepName: props.node.name,
                    dateCreated: props.node.finishedAt,
                    nodeName: props.node.name
                })
            )) ||
        [];
    return (
        <div className='white-box'>
            {artifacts.length === 0 && (
                <div className='row'>
                    <div className='columns small-12 text-center'>No artifacts</div>
                </div>
            )}
            {artifacts.length > 0 && props.archived && (
                <p>
                    <i className='fa fa-exclamation-triangle' /> Artifacts for archived workflows may be overwritten by a more recent workflow with the same name.
                </p>
            )}
            {artifacts.map(artifact => (
                <div className='row' key={artifact.name}>
                    <div className='columns small-1'>
                        <a href={artifact.downloadUrl}>
                            {' '}
                            <i className='fa fa-download' />
                        </a>
                    </div>
                    <div className='columns small-11'>
                        <span className='title'>{artifact.name}</span>
                        <div className='workflow-node-info__artifact-details'>
                            <span title={artifact.nodeName} className='muted'>
                                {artifact.nodeName}
                            </span>
                            <span title={artifact.path} className='muted'>
                                {artifact.path}
                            </span>
                            <span title={artifact.dateCreated} className='muted'>
                                <Timestamp date={artifact.dateCreated} />
                            </span>
                        </div>
                    </div>
                </div>
            ))}
        </div>
    );
}

export const WorkflowNodeInfo = (props: Props) => (
    <div className='workflow-node-info'>
        <Tabs
            navCenter={true}
            navTransparent={true}
            onTabSelected={props.onTabSelected}
            selectedTabKey={props.selectedTabKey}
            tabs={[
                {
                    title: 'SUMMARY',
                    key: 'summary',
                    content: (
                        <div>
                            <WorkflowNodeSummary {...props} />
                        </div>
                    )
                }
            ].concat(
                props.node.type !== 'Container'
                    ? [
                          {
                              title: 'CONTAINERS',
                              key: 'containers',
                              content: <WorkflowNodeContainers {...props} />
                          },
                          {
                              title: 'INPUTS/OUTPUTS',
                              key: 'inputs-outputs',
                              content: (
                                  <>
                                      <WorkflowNodeInputs {...props} />
                                      <WorkflowNodeOutputs {...props} />
                                  </>
                              )
                          }
                      ]
                    : []
            )}
        />
    </div>
);

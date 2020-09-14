import {Duration, Tabs, Ticker} from 'argo-ui';
import * as moment from 'moment';
import * as React from 'react';

import * as models from '../../../../models';
import {InlineTable} from '../../../shared/components/inline-table/inline-table';
import {Phase} from '../../../shared/components/phase';
import {Timestamp} from '../../../shared/components/timestamp';
import {ResourcesDuration} from '../../../shared/resources-duration';
import {services} from '../../../shared/services';
import {getResolvedTemplates} from '../../../shared/template-resolution';
import {EventsPanel} from '../events-panel';

require('./workflow-node-info.scss');

function nodeDuration(node: models.NodeStatus, now: moment.Moment) {
    const endTime = node.finishedAt ? moment(node.finishedAt) : now;
    return endTime.diff(moment(node.startedAt)) / 1000;
}

interface Props {
    node: models.NodeStatus;
    workflow: models.Workflow;
    links: models.Link[];
    archived: boolean;
    onShowContainerLogs: (nodeId: string, container: string) => any;
    onShowYaml?: (nodeId: string) => any;
}

const AttributeRow = (attr: {title: string; value: any}) => (
    <div className='row white-box__details-row' key={attr.title}>
        <div className='columns small-4'>{attr.title}</div>
        <div className='columns small-8'>{attr.value}</div>
    </div>
);
const AttributeRows = (props: {attributes: {title: string; value: any}[]}) => (
    <div>
        {props.attributes.map(attr => (
            <AttributeRow key={attr.title} {...attr} />
        ))}
    </div>
);

export const WorkflowNodeSummary = (props: Props) => {
    const attributes = [
        {title: 'NAME', value: props.node.name},
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
        {title: 'START TIME', value: <Timestamp date={props.node.startedAt} />},
        {title: 'END TIME', value: <Timestamp date={props.node.finishedAt} />},
        {
            title: 'DURATION',
            value: (
                <Ticker disabled={props.workflow.status && props.workflow.status.phase !== models.NODE_PHASE.RUNNING}>
                    {now => <Duration durationMs={nodeDuration(props.node, now)} />}
                </Ticker>
            )
        },
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
        attributes.splice(2, 0, {title: 'POD NAME', value: props.node.id}, {title: 'HOST NODE NAME', value: props.node.hostNodeName});
    }
    if (props.node.resourcesDuration) {
        attributes.push({
            title: 'RESOURCES DURATION',
            value: <ResourcesDuration resourcesDuration={props.node.resourcesDuration} />
        });
    }
    return (
        <div className='white-box'>
            <div className='white-box__details'>{<AttributeRows attributes={attributes} />}</div>
            <div>
                <button className='argo-button argo-button--base-o' onClick={() => props.onShowYaml && props.onShowYaml(props.node.id)}>
                    YAML
                </button>{' '}
                {props.node.type === 'Pod' && (
                    <button className='argo-button argo-button--base-o' onClick={() => props.onShowContainerLogs && props.onShowContainerLogs(props.node.id, 'main')}>
                        LOGS
                    </button>
                )}
                {props.links &&
                    props.links
                        .filter(link => link.scope === 'pod')
                        .map(link => (
                            <a
                                className='argo-button argo-button--base-o'
                                href={link.url.replace('${metadata.namespace}', props.workflow.metadata.namespace).replace('${metadata.name}', props.node.id)}>
                                <i className='fa fa-link' /> {link.name}
                            </a>
                        ))}
            </div>
        </div>
    );
};

export const WorkflowNodeInputs = (props: {inputs: models.Inputs}) => {
    const parameters = (props.inputs.parameters || []).map(artifact => ({
        title: artifact.name,
        value: artifact.value
    }));
    const artifacts = (props.inputs.artifacts || []).map(artifact => ({
        title: artifact.name,
        value: artifact.path
    }));
    return (
        <div className='white-box'>
            <div className='white-box__details'>
                {parameters.length > 0 && [
                    <div className='row white-box__details-row' key='title'>
                        <p>Parameters</p>
                    </div>,
                    <AttributeRows key='attrs' attributes={parameters} />
                ]}
                {artifacts.length > 0 && [
                    <div className='row white-box__details-row' key='title'>
                        <p>Input Artifacts</p>
                    </div>,
                    <AttributeRows key='attrs' attributes={artifacts} />
                ]}
            </div>
        </div>
    );
};

function hasEnv(container: models.kubernetes.Container | models.Sidecar | models.Script): container is models.kubernetes.Container | models.Sidecar {
    return (container as models.kubernetes.Container | models.Sidecar).env !== undefined;
}

export const WorkflowNodeContainer = (props: {
    nodeId: string;
    container: models.kubernetes.Container | models.Sidecar | models.Script;
    onShowContainerLogs: (nodeId: string, container: string) => any;
}) => {
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
                  value: <pre className='workflow-node-info__multi-line'>{(container.env || []).map(e => `${e.name}=${e.value}`).join('\n')}</pre>
              }
            : {title: 'ENV', value: <pre className='workflow-node-info__multi-line' />}
    ];
    return (
        <div className='white-box'>
            <div className='white-box__details'>{<AttributeRows attributes={attributes} />}</div>
            <div>
                <button className='argo-button argo-button--base-o' onClick={() => props.onShowContainerLogs && props.onShowContainerLogs(props.nodeId, container.name)}>
                    LOGS
                </button>
            </div>
        </div>
    );
};

export class WorkflowNodeContainers extends React.Component<Props, {selectedSidecar: string}> {
    constructor(props: Props) {
        super(props);
        this.state = {selectedSidecar: null};
    }

    public render() {
        const template = getResolvedTemplates(this.props.workflow, this.props.node);
        if (!template || (!template.container && !template.script)) {
            return (
                <div className='white-box'>
                    <div className='row'>
                        <div className='columns small-12 text-center'>No data to display</div>
                    </div>
                </div>
            );
        }
        const container =
            (this.state.selectedSidecar && template.sidecars && template.sidecars.find(item => item.name === this.state.selectedSidecar)) || template.container || template.script;
        return (
            <div className='workflow-node-info__containers'>
                {this.state.selectedSidecar && <i className='fa fa-angle-left workflow-node-info__sidecar-back' onClick={() => this.setState({selectedSidecar: null})} />}
                <WorkflowNodeContainer nodeId={this.props.node.id} container={container} onShowContainerLogs={this.props.onShowContainerLogs} />
                {!this.state.selectedSidecar && template.sidecars && template.sidecars.length > 0 && (
                    <div>
                        <p>SIDECARS:</p>
                        {template.sidecars.map(sidecar => (
                            <div className='workflow-node-info__sidecar' key={sidecar.name} onClick={() => this.setState({selectedSidecar: sidecar.name})}>
                                <span>{sidecar.name}</span> <i className='fa fa-angle-right' />
                            </div>
                        ))}
                    </div>
                )}
            </div>
        );
    }
}

export const WorkflowNodeArtifacts = (props: Props) => {
    const artifacts =
        (props.node.outputs &&
            props.node.outputs.artifacts &&
            props.node.outputs.artifacts.map(artifact =>
                Object.assign({}, artifact, {
                    downloadUrl: services.workflows.getArtifactDownloadUrl(props.workflow, props.node.id, artifact.name, props.archived),
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
                    <div className='columns small-12 text-center'>No data to display</div>
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
                            <i className='icon argo-icon-artifact' />
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
                            <span title={artifact.dateCreated.toString()} className='muted'>
                                <Timestamp date={artifact.dateCreated} />
                            </span>
                        </div>
                    </div>
                </div>
            ))}
        </div>
    );
};

export const WorkflowNodeInfo = (props: Props) => (
    <div className='workflow-node-info'>
        <Tabs
            navCenter={true}
            navTransparent={true}
            tabs={[
                {
                    title: 'SUMMARY',
                    key: 'summary',
                    content: (
                        <div>
                            <WorkflowNodeSummary {...props} />
                            {props.node.inputs && <WorkflowNodeInputs inputs={props.node.inputs} />}
                        </div>
                    )
                },
                props.node.type === 'Pod' && {
                    title: 'EVENTS',
                    key: 'events',
                    content: <EventsPanel namespace={props.workflow.metadata.namespace} kind='Pod' name={props.node.id} />
                },
                {
                    title: 'CONTAINERS',
                    key: 'containers',
                    content: <WorkflowNodeContainers {...props} />
                },
                {
                    title: 'OUTPUT ARTIFACTS',
                    key: 'artifacts',
                    content: <WorkflowNodeArtifacts {...props} />
                }
            ]}
        />
    </div>
);

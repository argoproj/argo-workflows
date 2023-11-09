import * as React from 'react';

import * as models from '../../../models';
import {Timestamp} from '../../shared/components/timestamp';
import {services} from '../../shared/services';

interface Props {
    workflow: models.Workflow;
    archived: boolean;
}

export function WorkflowArtifacts(props: Props) {
    const workflowStatusNodes = (props.workflow.status && props.workflow.status.nodes) || {};
    const artifacts =
        Object.keys(workflowStatusNodes)
            .map(nodeName => {
                const node = workflowStatusNodes[nodeName];
                const nodeOutputs = node.outputs || {artifacts: [] as models.Artifact[]};
                const items = nodeOutputs.artifacts || [];
                return items.map(item =>
                    Object.assign({}, item, {
                        downloadUrl: services.workflows.getArtifactDownloadUrl(props.workflow, node.id, item.name, props.archived, false),
                        stepName: node.name,
                        dateCreated: node.finishedAt,
                        nodeName
                    })
                );
            })
            .reduce((first, second) => first.concat(second), []) || [];
    if (artifacts.length === 0) {
        return (
            <div className='white-box'>
                <div className='row'>
                    <div className='columns small-12 text-center'>No data to display</div>
                </div>
            </div>
        );
    }
    return (
        <div className='white-box'>
            <div className='white-box__details'>
                {artifacts.map(artifact => (
                    <div className='row white-box__details-row' key={artifact.downloadUrl}>
                        <div className='columns small-2'>
                            <span>
                                <a href={artifact.downloadUrl}>
                                    {' '}
                                    <i className='fa fa-download' />
                                </a>{' '}
                                {artifact.name}
                            </span>
                        </div>
                        <div className='columns small-4'>{artifact.stepName}</div>
                        <div className='columns small-3'>{artifact.path}</div>
                        <div className='columns small-3'>
                            <Timestamp date={artifact.dateCreated} />
                        </div>
                    </div>
                ))}
            </div>
        </div>
    );
}

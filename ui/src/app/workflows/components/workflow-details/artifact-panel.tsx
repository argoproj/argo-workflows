import * as React from 'react';
import {Artifact, Workflow} from '../../../../models';
import {artifactKey} from '../../../shared/artifacts';
import {LinkButton} from '../../../shared/components/link-button';
import {services} from '../../../shared/services';

export const ArtifactPanel = ({
    workflow,
    artifact,
    archived
}: {
    workflow: Workflow;
    artifact: Artifact & {nodeId: string; artifactNameDiscriminator: string};
    archived?: boolean;
}) => {
    const downloadUrl = services.workflows.getArtifactDownloadUrl(workflow, artifact.nodeId, artifact.name, archived, artifact.artifactNameDiscriminator === 'input');

    const key = artifactKey(artifact)
        .split('/')
        .pop();

    return (
        <div style={{margin: 16, marginTop: 48}} className='white-box'>
            <h3>{artifact.name}</h3>
            <p>
                <LinkButton to={downloadUrl}>
                    <i className='fa fa-download' /> {key || 'Download'}
                </LinkButton>
            </p>
        </div>
    );
};

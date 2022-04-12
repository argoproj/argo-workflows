import * as React from 'react';
import {Artifact, Workflow} from '../../../../models';
import {InfoIcon} from '../../../shared/components/fa-icons';
import {LinkButton} from '../../../shared/components/link-button';
import {services} from '../../../shared/services';

export const ArtifactPanel = ({workflow, artifact, archived}: {workflow: Workflow; artifact: Artifact & {nodeId: string; input: boolean}; archived?: boolean}) => {
    const url = services.workflows.getArtifactDownloadUrl(workflow, artifact.nodeId, artifact.name, archived, artifact.input);
    return (
        <div style={{margin: 16, marginTop: 48}}>
            <div>
                <h3>{artifact.name}</h3>
                <p>{artifact.path}</p>
                {artifact.archive?.none ? (
                    <div className='white-box'>
                        <iframe src={url} frameBorder={0} width='100%' height={400} />
                    </div>
                ) : (
                    <p>
                        <InfoIcon /> Artifacts are viewable if they use "archive: none".
                    </p>
                )}
                <p>
                    <LinkButton to={url}>
                        <i className='fa fa-download' /> Download
                    </LinkButton>
                </p>
            </div>
            <p className='fa-pull-right'>
                <small>
                    <a href='https://github.com/argoproj/argo-workflows/issues/8324'>
                        <i className='fa fa-comment' /> Give feedback{' '}
                    </a>
                </small>
            </p>
        </div>
    );
};

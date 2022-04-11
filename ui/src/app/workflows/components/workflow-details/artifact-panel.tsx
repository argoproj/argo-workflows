import * as React from 'react';
import {Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {InfoIcon} from '../../../shared/components/fa-icons';
import {LinkButton} from '../../../shared/components/link-button';
import {services} from '../../../shared/services';
import {WorkflowDag} from '../workflow-dag/workflow-dag';

export const ArtifactPanel = ({workflow, selectedArtifact, archived}: {workflow: Workflow; selectedArtifact: string; archived?: boolean}) => {
    const ar = workflow.status.artifactRepositoryRef?.artifactRepository;

    const artifacts: {name: string; path?: string; url: string; archive?: {none?: any}}[] = [];

    Object.values(workflow.status.nodes).map(node => {
        return (node.inputs?.artifacts || [])
            .map(ia => ({isInput: true, ...ia}))
            .concat((node.outputs?.artifacts || []).map(oa => ({isInput: false, ...oa})))
            .map(a => ({...a, ...WorkflowDag.artifactDescription(a, ar)}))
            .filter(({id}) => id === selectedArtifact)
            .map(d => ({url: uiUrl(services.workflows.getArtifactDownloadUrl(workflow, node.id, d.name, archived, d.isInput)), ...d}))
            .forEach(d => artifacts.push(d));
    });

    if (artifacts.length === 0) {
        return <ErrorNotice error={new Error('artifact not found')} />;
    }

    const art = artifacts[artifacts.length - 1];

    return (
        <div style={{margin: 16, marginTop: 48}}>
            <div>
                <h3>{art.name}</h3>
                <p>{art.path}</p>
                {art.archive?.none ? (
                    <div className='white-box'>
                        <iframe src={art.url} frameBorder={0} width='100%' height={400} />
                    </div>
                ) : (
                    <p>
                        <InfoIcon /> Artifacts are viewable if they use "archive: none".
                    </p>
                )}
                <p>
                    <LinkButton to={art.url}>
                        <i className='fa fa-download' /> Download
                    </LinkButton>
                </p>
            </div>
        </div>
    );
};

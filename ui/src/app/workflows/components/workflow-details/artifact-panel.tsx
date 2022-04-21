import * as React from 'react';
import {useEffect, useState} from 'react';
import {Artifact, Workflow} from '../../../../models';
import {uiUrl} from '../../../shared/base';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {LinkButton} from '../../../shared/components/link-button';
import {services} from '../../../shared/services';
import {ArtifactDescription} from '../../../shared/services/artifact-service';

require('./artifact-panel.scss');

function formatBytes(bytes: number) {
    const sizes = ['bytes', 'KB', 'MB', 'GB', 'TB'];
    if (bytes === 0) {
        return '0 bytes';
    }
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return Math.round(bytes / Math.pow(1024, i)) + ' ' + sizes[i];
}

const ItemViewer = ({src}: {src: string}) => (
    <div className='white-box'>
        <iframe src={src} frameBorder={0} width='100%' height={400} />
    </div>
);

export const ArtifactPanel = ({workflow, artifact, archived}: {workflow: Workflow; artifact: Artifact & {nodeId: string; artifactDiscrim: string}; archived?: boolean}) => {
    const [error, setError] = useState<Error>();
    const [description, setDescription] = useState<ArtifactDescription>();
    const [selectedItem, setSelectedItem] = useState<string>();
    const [showAnyway, setShowAnyway] = useState<boolean>();

    useEffect(() => {
        setDescription(null);
        setError(null);
        setSelectedItem(null);
        setShowAnyway(false);
        services.artifacts
            .getArtifactDescription(workflow.metadata.namespace, workflow.metadata.name, artifact.nodeId, artifact.artifactDiscrim, artifact.name)
            .then(setDescription)
            .catch(setError);
    }, [workflow.metadata.namespace, workflow.metadata.name, artifact.nodeId, artifact.artifactDiscrim, artifact.name]);

    useEffect(() => {
        setSelectedItem((description?.items || []).find(item => showAnyway || item.contentType?.startsWith('text/'))?.filename);
    }, [description, showAnyway]);

    const idDiscrim = archived ? 'uid' : 'name';
    const id = archived ? workflow.metadata.uid : workflow.metadata.name;

    const downloadUrl = uiUrl(`workflow-artifacts/v2/artifacts/${workflow.metadata.namespace}/${idDiscrim}/${id}/${artifact.nodeId}/${artifact.artifactDiscrim}/${artifact.name}`);
    const itemDownloadUrl = (item: string) =>
        uiUrl(`workflow-artifacts/v2/artifacts/${workflow.metadata.namespace}/${idDiscrim}/${id}/${artifact.nodeId}/${artifact.artifactDiscrim}/${artifact.name}/${item}`);

    const filename = description?.filename;

    return (
        <div style={{margin: 16, marginTop: 48}}>
            <h3>{filename || artifact.name}</h3>
            {error && <ErrorNotice error={error} />}
            {description?.items && (
                <div className='white-box'>
                    {description.items.map(item => (
                        <div className='row' key={item.filename}>
                            <div className='columns small-8'>
                                <a href={itemDownloadUrl(item.filename)} target='_blank'>
                                    <i className='fa fa-external-link-alt' />
                                </a>{' '}
                                <a onClick={() => setSelectedItem(item.filename)} className={item.filename === selectedItem && 'selectedItem'}>
                                    {item.filename}
                                </a>
                            </div>
                            <div className='columns small-4'>
                                <a href={itemDownloadUrl(item.filename)}>
                                    <i className='fa fa-download' />
                                </a>{' '}
                                <span className=' muted'>{formatBytes(item.size)}</span>
                            </div>
                        </div>
                    ))}
                </div>
            )}
            {selectedItem ? (
                <ItemViewer src={itemDownloadUrl(selectedItem)} />
            ) : description?.contentType?.startsWith('text/') || showAnyway ? (
                <ItemViewer src={downloadUrl} />
            ) : (
                <p>
                    Does not appear to be a text file, <a onClick={() => setShowAnyway(true)}>show anyway</a>.
                </p>
            )}
            <p>
                <LinkButton to={downloadUrl}>
                    <i className='fa fa-download' /> {filename || 'Download'}
                </LinkButton>
            </p>
            <p className='fa-pull-right muted'>
                <a href='https://github.com/argoproj/argo-workflows/issues/8324'>
                    <i className='fa fa-comment' /> Give feedback
                </a>
            </p>
        </div>
    );
};

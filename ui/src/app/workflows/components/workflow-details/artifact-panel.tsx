import * as React from 'react';
import {useEffect, useState} from 'react';
import MonacoEditor from 'react-monaco-editor';
import {Artifact, ArtifactRepository, Workflow} from '../../../../models';
import {artifactKey, artifactURN} from '../../../shared/artifacts';
import ErrorBoundary from '../../../shared/components/error-boundary';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {FirstTimeUserPanel} from '../../../shared/components/first-time-user-panel';
import {GiveFeedbackLink} from '../../../shared/components/give-feedback-link';
import {LinkButton} from '../../../shared/components/link-button';
import {useCollectEvent} from '../../../shared/components/use-collect-event';
import {services} from '../../../shared/services';
import requests from '../../../shared/services/requests';

export const ArtifactPanel = ({
    workflow,
    artifact,
    archived,
    artifactRepository
}: {
    workflow: Workflow;
    artifact: Artifact & {nodeId: string; artifactNameDiscriminator: string};
    archived?: boolean;
    artifactRepository: ArtifactRepository;
}) => {
    const input = artifact.artifactNameDiscriminator === 'input';
    const downloadUrl = services.workflows.getArtifactDownloadUrl(workflow, artifact.nodeId, artifact.name, archived, input);

    const urn = artifactURN(artifact, artifactRepository);
    const key = artifactKey(artifact);
    const isDir = key.endsWith('/');
    const filename = key.split('/').pop();
    const ext = filename.split('.').pop();

    const [showExtension, setShowExtension] = useState(false);
    const [error, setError] = useState<Error>();
    const [object, setObject] = useState<any>();
    const [httpStatus, setHTTPStatus] = useState(200);

    const tgz = !input && !artifact.archive?.none; // the key can be wrong about the file type
    const supported = !tgz && (isDir || ['gif', 'jpg', 'jpeg', 'json', 'html', 'png', 'txt'].includes(ext));
    useEffect(() => setShowExtension(supported), [downloadUrl, ext]);

    useEffect(() => {
        setObject(null);
        setError(null);
        if (ext === 'json') {
            // show the object below
            requests
                .get(services.workflows.artifactPath(workflow, artifact.nodeId, artifact.name, archived, input))
                .then(r => {setHTTPStatus(r.status); r.text})
                .then(setObject)
                .catch(e => {setError(e); setHTTPStatus(e.response.status) });
        } else if (ext == 'tgz') {
            setHTTPStatus(200); // since we're not downloading the file yet, reset httpStatus back to success 
        } else {
            // even though we include the file in an iframe, if we download it first here we can prevent showing the download button if the status is failed
            requests
                .get(services.workflows.artifactPath(workflow, artifact.nodeId, artifact.name, archived, input))
                .then(r => {setHTTPStatus(r.status)})
                .catch(e => {setHTTPStatus(e.response.status)})
        } 
    }, [downloadUrl]);
    useCollectEvent('openedArtifactPanel');

    return (
        <div style={{margin: 16, marginTop: 48}}>
            <FirstTimeUserPanel
                id='ArtifactPanel'
                explanation={
                    'This panel shows your workflow artifacts. This will work for any artifact that the argo-server can access. ' +
                    'That typically means you used access/key to connect to the repository, rather than annotations like eks.amazonaws.com/role-arn.'
                }>
                <ErrorBoundary>
                    <div className='white-box'>
                        <h3>{artifact.name}</h3>
                        <p>
                            <small>{urn}</small>
                        </p>
                        {error && <ErrorNotice error={error} />}
                        {(httpStatus >= 400 && httpStatus < 500) ? (
                            <p>File not found</p>
                        ) : showExtension ? (
                            <ViewBox>
                                {object ? (
                                    <MonacoEditor
                                        value={object}
                                        language='json'
                                        height='500px'
                                        options={{
                                            readOnly: true,
                                            minimap: {enabled: false},
                                            renderIndentGuides: true
                                        }}
                                    />
                                ) : (
                                    <iframe src={downloadUrl} style={{width: '100%', height: '500px', border: 'none'}} />
                                )}
                            </ViewBox>
                        ) : tgz ? (
                            <p>Artifact cannot be shown because it is a tgz.</p>
                        ) : (
                            <p>
                                Unknown extension "{ext}", <a onClick={() => setShowExtension(true)}>show anyway</a>.
                            </p>
                        )}

                        {(httpStatus >= 200 && httpStatus < 300) && (
                            <p style={{marginTop: 10}}>
                                <LinkButton to={downloadUrl}>
                                    <i className='fa fa-download' /> {filename || 'Download'}
                                </LinkButton>
                            </p>
                        )}
                        <GiveFeedbackLink href='https://github.com/argoproj/argo-workflows/issues/7743' />
                    </div>
                </ErrorBoundary>
            </FirstTimeUserPanel>
        </div>
    );
};

const ViewBox = ({children}: {children: React.ReactElement}) => <div style={{border: 'solid 1px #ddd', padding: 10}}>{children}</div>;

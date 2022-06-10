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

    const [show, setShow] = useState(false);
    const [errorRenamed, setError] = useState<Error>();
    const [object, setObject] = useState<any>();
    const [httpStatus, setHTTPStatus] = useState(200);
    const [showDownloadLink, setShowDownloadLink] = useState(true);

    const tgz = !input && !artifact.archive?.none; // the key can be wrong about the file type
    const supported = !tgz && (isDir || ['gif', 'jpg', 'jpeg', 'json', 'html', 'png', 'txt'].includes(ext));
    useEffect(() => setShow(supported), [downloadUrl, ext]);

    useEffect(() => {
        setObject(null);
        setError(null);
        setShowDownloadLink(true);
        if (ext === 'json') {
            requests
                .get(services.workflows.artifactPath(workflow, artifact.nodeId, artifact.name, archived, input))
                .then(r => r.text)
                .then(setObject)
                .catch(setError);
        } else {
            requests
                .get(services.workflows.artifactPath(workflow, artifact.nodeId, artifact.name, archived, input))
                .then(function onResult(res) {
                    console.log(res);
                    setHTTPStatus(res.status);
                  }, function onError(err) {
                      console.log(err.response);
                      setHTTPStatus(err.response.status);
                      setShowDownloadLink(false);
                  });
        }
    }, [downloadUrl]);
    useCollectEvent('openedArtifactPanel');

    //const internalServerError =  (errorRenamed.status == 500);

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
                        {errorRenamed && <ErrorNotice error={errorRenamed} />}
                        { (httpStatus == 500) ? ( //todo: change to 404
                            <p>Artifact has been deleted.</p>
                        ) :  show ? (
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
                                Unknown extension "{ext}", <a onClick={() => setShow(true)}>show anyway</a>.
                            </p>
                        )}
                        
                        {showDownloadLink && (
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

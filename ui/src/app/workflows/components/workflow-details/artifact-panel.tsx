import * as React from 'react';
import {useEffect, useState} from 'react';
import MonacoEditor from 'react-monaco-editor';
import {Artifact, Workflow} from '../../../../models';
import {artifactKey} from '../../../shared/artifacts';
import ErrorBoundary from '../../../shared/components/error-boundary';
import {ErrorNotice} from '../../../shared/components/error-notice';
import {FirstTimeUserPanel} from '../../../shared/components/first-time-user-panel';
import {GiveFeedbackLink} from '../../../shared/components/give-feedback-link';
import {LinkButton} from '../../../shared/components/link-button';
import requests from '../../../shared/services/requests';

export const ArtifactPanel = ({
    workflow,
    artifact,
    archived
}: {
    workflow: Workflow;
    artifact: Artifact & {nodeId: string; artifactNameDiscriminator: string};
    archived?: boolean;
}) => {
    const archiveDiscriminator = archived ? 'archived-workflows' : 'workflows';
    const downloadUrl = `artifact-files/${workflow.metadata.namespace}/${archiveDiscriminator}/${workflow.metadata.name}/${artifact.nodeId}/outputs/${artifact.name}`;

    const key = artifactKey(artifact);
    const isDir = key.endsWith('/');
    const filename = key
        .replace(/\/$/, '')
        .split('/')
        .pop();
    const ext = key.split('.').pop();

    const [show, setShow] = useState(false);
    const [error, setError] = useState<Error>();
    const [object, setObject] = useState<any>();

    useEffect(() => setShow(isDir || ['gif', 'jpg', 'jpeg', 'json', 'html', 'png', 'txt'].includes(ext)), [key]);

    useEffect(() => {
        setObject(null);
        setError(null);
        if (ext === 'json') {
            requests
                .get(downloadUrl)
                .then(r => r.text)
                .then(setObject)
                .catch(setError);
        }
    }, [downloadUrl]);

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
                        {error && <ErrorNotice error={error} />}
                        {show ? (
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
                        ) : (
                            <p>
                                Unknown extension "{ext}", <a onClick={() => setShow(true)}>show anyway</a>.
                            </p>
                        )}

                        <p style={{marginTop: 10}}>
                            <LinkButton to={downloadUrl}>
                                <i className='fa fa-download' /> {filename || 'Download'}
                            </LinkButton>
                        </p>
                        <GiveFeedbackLink href='https://github.com/argoproj/argo-workflows/issues/7743' />
                    </div>
                </ErrorBoundary>
            </FirstTimeUserPanel>
        </div>
    );
};

const ViewBox = ({children}: {children: React.ReactElement}) => <div style={{border: 'solid 1px #ddd', padding: 10}}>{children}</div>;

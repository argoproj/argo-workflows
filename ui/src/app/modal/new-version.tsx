import * as React from 'react';
import {Button} from '../shared/components/button';
import {LinkButton} from '../shared/components/link-button';
import {Modal} from './modal';

export const NewVersion = ({version, dismiss}: {version: string; dismiss: () => void}) => {
    return (
        <Modal
            title={
                <>
                    {' '}
                    It looks like <b>{version}</b> has just been installed!
                </>
            }>
            <h5>What's new?</h5>
            <div style={{margin: 'auto'}}>
                <ul>
                    <li>Python and Java SDKs for writing workflows without YAML.</li>
                    <li>Use HTTP template to interact with third-party systems.</li>
                    <li>Use container set template and Emissary to run workflows faster and cheaper.</li>
                    <li>Visual ArgoLogs Dataflow pipelines.</li>
                    <li>Delayed deletion of pods, to help debug issues.</li>
                    <li>Use the new data template to fan-out workflows based on bucket contents.</li>
                </ul>
            </div>
            <div>
                <LinkButton to='https://blog.argoproj.io'>Read more about new features on our blog</LinkButton>
                <LinkButton to='https://github.com/argoproj/argo-workflows/blob/master/CHANGELOG.md'>Read the full changelog</LinkButton>
                <Button onClick={() => dismiss}>Close</Button>
            </div>
        </Modal>
    );
};

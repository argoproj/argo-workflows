import * as React from 'react';
import {Modal} from '../shared/components/modal/modal';

require('./new-version-modal.scss');

const links = {
    'v3.0': 'https://blog.argoproj.io/argo-workflows-v3-0-4d0b69f15a6e?utm_source=argo-ui',
    'v3.1': 'https://blog.argoproj.io/argo-workflows-v3-1-is-coming-1fb1c1091324?utm_source=argo-ui',
    'v3.2': 'https://blog.argoproj.io/argo-workflows-v3-2-af780a99b362?utm_source=argo-ui'
};

export const NewVersionModal = ({version, dismiss}: {version: string; dismiss: () => void}) => {
    return (
        <Modal dismiss={dismiss}>
            <div className='new-version-modal-banner'>
                <i className='fa fa-arrow-circle-up' />{' '}
            </div>
            <h4 className='new-version-modal-title'>
                It looks like <b>{version}</b> has just been installed!
            </h4>
            <p>Recent changes:</p>
            <ul className='new-version-modal-bullets'>
                <li>
                    Writing workflows without YAML using{' '}
                    <a href={links['v3.2']} target='_blank'>
                        {' '}
                        Python and Java SDKs
                    </a>
                    .
                </li>
                <li>
                    Visualize{' '}
                    <a href={links['v3.2']} target='_blank'>
                        ArgoLabs Dataflow pipelines
                    </a>
                    .
                </li>
                <li>
                    Interact with third-party systems using{' '}
                    <a href={links['v3.2']} target='_blank'>
                        HTTP template
                    </a>
                    .
                </li>
                <li>
                    Run workflows faster and cheaper using{' '}
                    <a href={links['v3.1']} target='_blank'>
                        container set template and Emissary executor
                    </a>
                    .
                </li>
                <li>
                    Run fan-out workflows based on bucket contents using{' '}
                    <a href={links['v3.1']} target='_blank'>
                        data templates
                    </a>
                    .
                </li>
                <li>
                    Complex and dynamic templating using{' '}
                    <a href={links['v3.1']} target='_blank'>
                        expression tag templates
                    </a>
                    .
                </li>
                <li>
                    Embed widgets in your own apps with{' '}
                    <a href={links['v3.0']} target='_blank'>
                        widgets
                    </a>
                    .
                </li>
            </ul>
            <p>
                <a href='https://github.com/argoproj/argo-workflows/blob/master/CHANGELOG.md' target='_blank'>
                    Changelog
                </a>
            </p>
        </Modal>
    );
};

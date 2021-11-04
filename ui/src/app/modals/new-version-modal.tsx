import * as React from 'react';
import {Modal} from '../shared/components/modal/modal';

require('./new-version-modal.scss');
/**
 * The intention of this modal is to encourage update of new features.
 */
export const NewVersionModal = ({version, dismiss}: {version: string; dismiss: () => void}) => {
    return (
        <Modal dismiss={dismiss}>
            <div className='new-version-modal-banner'>
                <i className='fa fa-arrow-circle-up' />{' '}
            </div>
            <h4 className='new-version-modal-title'>
                It looks like <b>{version}</b> has just been installed!
            </h4>
            <h5>
                <a href='https://blog.argoproj.io/argo-workflows-v3-2-af780a99b362?utm_source=argo-ui'>v3.2</a>
            </h5>
            <ul className='new-version-modal-bullets'>
                <li>
                    Writing workflows <b>without YAML</b> using Python and Java SDKs.
                </li>
                <li>Visualize ArgoLabs Dataflow pipelines.</li>
                <li>Interact with third-party systems using HTTP template.</li>
            </ul>
            <h5>
                <a href='https://blog.argoproj.io/argo-workflows-v3-1-is-coming-1fb1c1091324?utm_source=argo-ui'>v3.1</a>
            </h5>
            <ul className='new-version-modal-bullets'>
                <li>
                    Run workflows <b>faster and cheaper</b> using container set template and Emissary executor.
                </li>
                <li>Run fan-out workflows based on bucket contents using data templates.</li>
                <li>Complex and dynamic templating using expression tag templates.</li>
            </ul>
        </Modal>
    );
};

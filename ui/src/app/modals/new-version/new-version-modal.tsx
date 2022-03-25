import * as React from 'react';
import {Modal} from '../../shared/components/modal/modal';
import {SurveyButton} from '../../shared/components/survey-button';

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
            <h5>v3.3</h5>
            <ul className='new-version-modal-bullets'>
                <li>
                    Write your own types of step with{' '}
                    <a href='https://argoproj.github.io/argo-workflows/plugins/?utm_source=argo-ui' target='_blank'>
                        plugin templates
                    </a>
                    .
                </li>
                <li>Connect with the workflow using lifecycle hooks.</li>
                <li>Pause steps in your workflow with Debug pause.</li>
            </ul>
            <p>
                <a href='https://blog.argoproj.io/whats-new-in-argo-workflows-v3-3-dd051d2f1c7?utm_source=argo-ui' target='_blank'>
                    Learn more
                </a>
            </p>
            <h5>v3.2</h5>
            <ul className='new-version-modal-bullets'>
                <li>
                    Writing workflows <b>without YAML</b> using{' '}
                    <a href='https://argoproj.github.io/argo-workflows/client-libraries/?utm_source=argo-ui' target='_blank'>
                        Python and Java SDKs
                    </a>
                    .
                </li>
                <li>
                    Visualize and debug{' '}
                    <a href='https://github.com/argoproj-labs/argo-dataflow' target='_blank'>
                        Dataflow pipelines
                    </a>
                    .
                </li>
                <li>
                    Interact with third-party systems using{' '}
                    <a href='https://argoproj.github.io/argo-workflows/http-template/?utm_source=argo-ui' target='_blank'>
                        HTTP template
                    </a>
                    .
                </li>
            </ul>
            <p>
                <a href='https://blog.argoproj.io/argo-workflows-v3-2-af780a99b362?utm_source=argo-ui' target='_blank'>
                    Learn more
                </a>
            </p>
            <h5>v3.1</h5>
            <ul className='new-version-modal-bullets'>
                <li>
                    Run workflows <b>faster and cheaper</b>{' '}
                    <a href='https://argoproj.github.io/argo-workflows/container-set-template/?utm_source=argo-ui' target='_blank'>
                        using container set template and Emissary executor
                    </a>
                    .
                </li>
                <li>
                    Run fan-out workflows based on bucket contents using{' '}
                    <a href='https://argoproj.github.io/argo-workflows/data-sourcing-and-transformation/?utm_source=argo-ui' target='_blank'>
                        data templates
                    </a>
                    .
                </li>
                <li>
                    Complex and dynamic templating using{' '}
                    <a href='https://argoproj.github.io/argo-workflows/variables/?utm_source=argo-ui#expression' target='_blank'>
                        expression tag templates
                    </a>
                    .
                </li>
            </ul>
            <p>
                <a href='https://blog.argoproj.io/argo-workflows-v3-1-is-coming-1fb1c1091324?utm_source=argo-ui' target='_blank'>
                    Learn more
                </a>
            </p>
            <p className='new-version-modal-footer'>
                <SurveyButton />
            </p>
        </Modal>
    );
};

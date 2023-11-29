import * as React from 'react';
import {BigButton} from '../../shared/components/big-button';
import {Icon} from '../../shared/components/icon';
import {Modal} from '../../shared/components/modal/modal';
import {SurveyButton} from '../../shared/components/survey-button';

/**
 * The intention of this modal is to:
 * (a) help us understand what our users are using workflows for
 * (b) provide them with targeted support and docs
 */
export const FirstTimeUserModal = ({dismiss}: {dismiss: () => void}) => (
    <Modal dismiss={dismiss}>
        <h3 style={{textAlign: 'center'}}>Tell us what you want to use Argo for - we&apos;ll tell you how to do it. </h3>
        <div style={{textAlign: 'center'}}>
            {[
                {key: 'machine-learning', icon: 'brain', title: 'Machine Learning'},
                {key: 'data-processing', icon: 'database', title: 'Data Processing'},
                {key: 'stream-processing', icon: 'stream', title: 'Stream Processing'},
                {key: 'ci-cd', icon: 'sync-alt', title: 'CI/CD'},
                {key: 'infrastructure-automation', icon: 'network-wired', title: 'Infrastructure Automation'},
                {key: 'other', icon: 'question-circle', title: 'Other...'}
            ].map(({key, icon, title}) => (
                <BigButton key={key} title={title} icon={icon as Icon} href={`https://argoproj.github.io/argo-workflows/use-cases/${key}?utm_source=argo-ui`} />
            ))}
        </div>
        <p style={{textAlign: 'center', paddingTop: 20}}>
            <SurveyButton />
        </p>
    </Modal>
);

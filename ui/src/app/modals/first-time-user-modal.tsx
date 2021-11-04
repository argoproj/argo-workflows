import * as React from 'react';
import {BigButton} from '../shared/components/big-button';
import {Icon} from '../shared/components/icon';
import {Modal} from '../shared/components/modal/modal';

export const FirstTimeUserModal = ({dismiss}: {dismiss: () => void}) => (
    <Modal dismiss={dismiss}>
        <h3 style={{textAlign: 'center'}}>Tell us what you want to use Argo for - we'll tell you how to do it. </h3>
        <div style={{textAlign: 'center'}}>
            {[
                {key: 'machine-learning', icon: 'brain', title: 'Machine Learning'},
                {key: 'ci-cd', icon: 'sync-alt', title: 'CI/CD'},
                {key: 'data-processing', icon: 'database', title: 'Data Processing'},
                {key: 'etl', icon: 'cog', title: 'ETL'},
                {key: 'batch-processing', icon: 'clock', title: 'Batch Processing'},
                {key: 'infrastructure-automation', icon: 'network-wired', title: 'Infrastructure Automation'},
                {key: 'anomaly-detection', icon: 'search', title: 'Anomaly Detection'},
                {key: 'operation-analytics', icon: 'chart-line', title: 'Operational Analytics'},
                {key: 'other', icon: 'question-circle', title: 'Other...'}
            ].map(({key, icon, title}) => (
                <BigButton key={key} title={title} icon={icon as Icon} href={`https://argoproj.github.io/argo-workflows/use-cases/{key}?utm_source=argo-ui`} />
            ))}
        </div>
    </Modal>
);

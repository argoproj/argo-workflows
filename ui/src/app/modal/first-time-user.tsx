import * as React from 'react';
import {AwesomeButtons} from './awesome-buttons';
import {Modal} from './modal';

export const FirstTimeUser = ({dismiss}: {dismiss: () => void}) => (
    <Modal title={`Tell us what you want to use Argo for - we'll tell you how to do it.`}>
        <AwesomeButtons
            options={{
                'brain': 'Machine Learning',
                'sync-alt': 'CI/CD',
                'database': 'Data Processing',
                'table': 'ETL',
                'clock': 'Batch Processing',
                'network-wired': 'Infrastructure Automation',
                'question-circle': 'Something else...'
            }}
            dismiss={dismiss}
        />
    </Modal>
);

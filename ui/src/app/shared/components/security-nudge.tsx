import * as React from 'react';
import {ReactNode} from 'react';
import {Nudge} from './nudge';

export const SecurityNudge = (props: {children: ReactNode}) => (
    <Nudge key='security-nudge'>
        <i className='fa fa-lock-open status-icon--failed' /> {props.children} <a href='https://argoproj.github.io/argo-workflows/workflow-pod-security-context/'>Learn more</a>
    </Nudge>
);

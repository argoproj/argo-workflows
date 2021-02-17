import * as React from 'react';
import {ReactNode} from 'react';
import {Nudge} from './nudge';

export const CostOptimisationNudge = (props: {name: string; children: ReactNode}) => (
    <Nudge key={'cost-optimization-nudge/' + props.name}>
        <i className='fa fa-money-bill-alt status-icon--pending' /> {props.children} <a href='https://argoproj.github.io/argo-workflows/cost-optimisation/'>Learn more</a>
    </Nudge>
);

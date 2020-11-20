import * as React from 'react';
import {CSSProperties} from 'react';
import {ErrorNotice} from './error-notice';
import {Notice} from './notice';
import {PhaseIcon} from './phase-icon';

export type Status = Error | 'Pending' | 'Running' | 'Succeeded' | null;

export const StatusNotice = (props: {style?: CSSProperties; status: Status}) =>
    typeof props.status === 'undefined' ? null : typeof props.status === 'object' ? (
        <ErrorNotice key='notice' error={props.status} style={props.style} />
    ) : (
        <Notice style={props.style} key='notice'>
            <PhaseIcon value={props.status} /> {props.status}
        </Notice>
    );

import * as React from 'react';
import {CSSProperties} from 'react';
import {Notice} from './notice';
import {PhaseIcon} from './phase-icon';

// Display an error notice.
// If the error was a HTTP error (i.e. from super-agent), rather than just an unhelpful "Internal Server Error",
// it will display any message in the body.
export const ErrorNotice = (props: {style?: CSSProperties; error: Error & {response?: {body: {message?: string}}}}) => (
    <Notice style={props.style}>
        <PhaseIcon value='Error' /> {props.error.message || 'Unknown error. Open your browser error console for more information.'}
        {props.error.response && props.error.response.body && props.error.response.body.message && ': ' + props.error.response.body.message}
    </Notice>
);

import * as React from 'react';
import {Notice} from './notice';

// Display an error notice.
// If the error was a HTTP error (i.e. from super-agent), rather than just an unhelpful "Internal Server Error",
// it will display any message in the body.
export const ErrorNotice = (props: {error: Error & {response?: {body: {message?: string}}}}) => (
    <Notice>
        <i className='fa fa-exclamation-triangle status-icon--failed' />
        {props.error.message}
        {props.error.response && props.error.response.body && props.error.response.body.message && ': ' + props.error.response.body.message}
    </Notice>
);

import * as React from 'react';
import {CSSProperties, useEffect, useState} from 'react';
import {Notice} from './notice';
import {PhaseIcon} from './phase-icon';

// Display an error notice.
// If the error was a HTTP error (i.e. from super-agent), rather than just an unhelpful "Internal Server Error",
// it will display any message in the body.
export const ErrorNotice = (props: {style?: CSSProperties; error: Error & {response?: {body: {message?: string}}}; onReload?: () => void; reloadAfterSeconds?: number}) => {
    // This timer code is based on https://stackoverflow.com/questions/57137094/implementing-a-countdown-timer-in-react-with-hooks
    const reloadAfterSeconds = props.reloadAfterSeconds || 120;
    const reload = props.onReload || document.location.reload;
    const [timeLeft, setTimeLeft] = useState(reloadAfterSeconds);
    const canAutoReload = reload !== document.location.reload; // we cannot automatically call `document.location.reload`
    if (canAutoReload) {
        useEffect(() => {
            if (!timeLeft) {
                reload();
                setTimeLeft(reloadAfterSeconds);
            }
            const intervalId = setInterval(() => {
                setTimeLeft(timeLeft - 1);
            }, 1000);
            return () => clearInterval(intervalId);
        }, [timeLeft]);
    }
    return (
        <Notice style={props.style}>
            <PhaseIcon value='Error' /> {props.error.message || 'Unknown error. Open your browser error console for more information.'}
            {props.error.response && props.error.response.body && props.error.response.body.message && ': ' + props.error.response.body.message}:{' '}
            <a onClick={() => reload()}>
                <i className='fa fa-redo' /> Reload
            </a>{' '}
            {canAutoReload && `${timeLeft}s`}
        </Notice>
    );
};

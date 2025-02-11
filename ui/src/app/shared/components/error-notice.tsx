import * as React from 'react';
import {CSSProperties, useEffect, useState} from 'react';
import {Notice} from './notice';
import {PhaseIcon} from './phase-icon';

// Display an error notice.
// If the error was a HTTP error (i.e. from super-agent), rather than just an unhelpful "Internal Server Error",
// it will display any message in the body.
export const ErrorNotice = (props: {style?: CSSProperties; error: Error & {response?: {body: {message?: string}}}; onReload?: () => void; reloadAfterSeconds?: number}) => {
    if (!props.error) {
        return null;
    }
    const [error, setError] = useState(() => props.error); // allow us to close the error panel - in case it does not get automatically closed

    useEffect(() => {
        setError(props.error);
    }, [props.error]);

    // This timer code is based on https://stackoverflow.com/questions/57137094/implementing-a-countdown-timer-in-react-with-hooks
    const reloadAfterSeconds = props.reloadAfterSeconds || 120;
    const reload = props.onReload;
    const [timeLeft, setTimeLeft] = useState(reloadAfterSeconds);
    // we cannot automatically call `document.location.reload`
    if (reload) {
        useEffect(() => {
            if (!error) {
                return;
            }
            if (!timeLeft) {
                reload();
                setTimeLeft(reloadAfterSeconds);
            }
            const intervalId = setInterval(() => {
                setTimeLeft(timeLeft - 1);
            }, 1000);
            return () => clearInterval(intervalId);
        }, [timeLeft, error]);
    }
    if (!error) {
        return null;
    }
    return (
        <Notice {...props.style}>
            <span>
                <PhaseIcon value='Error' /> {error.message || 'Unknown error. Open your browser error console for more information.'}
                {error.response && error.response.body && error.response.body.message && ': ' + error.response.body.message}
            </span>
            {reload && (
                <span>
                    <a onClick={() => reload()}>
                        <i className='fa fa-redo' /> Reload
                    </a>{' '}
                    {timeLeft}s
                </span>
            )}
            <span className='fa-pull-right'>
                <a onClick={() => setError(null)}>
                    <i className='fa fa-times' />
                </a>
            </span>
        </Notice>
    );
};

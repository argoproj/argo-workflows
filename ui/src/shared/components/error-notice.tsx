import * as React from 'react';
import {CSSProperties, useEffect, useState} from 'react';

import {getErrorMessage, HttpError} from '../errors';
import {Notice} from './notice';
import {PhaseIcon} from './phase-icon';

// Display an error notice.
// If the error was a HTTP error (i.e. from super-agent), rather than just an unhelpful "Internal Server Error",
// it will display any message in the body.
export function ErrorNotice(props: {style?: CSSProperties; error: HttpError; onReload?: () => void; reloadAfterSeconds?: number}) {
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
                <PhaseIcon value='Error' /> {getErrorMessage(error)}
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
}

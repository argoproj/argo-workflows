import {ErrorInfo} from 'react';
import * as React from 'react';

interface Props {
    error?: Error & {response?: any};
    errorInfo?: ErrorInfo;
}

export function ErrorPanel(props: Props) {
    return (
        <div className='white-box'>
            <h3>
                <i className='fa fa-skull status-icon--failed' /> {props.error.message}
            </h3>
            <p>
                <i className='fa fa-redo' /> <a href='javascript:document.location.reload();'>Reload this page</a> to try again.
            </p>
            {props.error.response && (
                <>
                    {props.error.response.req && (
                        <>
                            <h5>Request</h5>
                            <pre>
                                {props.error.response.req.method} {props.error.response.req.url}
                            </pre>
                        </>
                    )}
                    <>
                        <h5>Response</h5>
                        <pre>HTTP {props.error.response.status}</pre>
                        {props.error.response.body && <pre>{JSON.stringify(props.error.response.body, null, 2)}</pre>}
                    </>
                </>
            )}
            <h5>Stack Trace</h5>
            <pre>{props.error.stack}</pre>
            {props.errorInfo && (
                <>
                    <h5>Component Stack</h5>
                    <pre>{props.errorInfo.componentStack}</pre>
                </>
            )}
        </div>
    );
}

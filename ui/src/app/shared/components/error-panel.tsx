import {ErrorInfo} from 'react';
import * as React from 'react';

interface Props {
    error?: Error & {response?: any};
    errorInfo?: ErrorInfo;
}

export class ErrorPanel extends React.Component<Props> {
    constructor(props: Props) {
        super(props);
    }

    public render() {
        return (
            <div className='white-box'>
                <h3>
                    <i className='fa fa-skull status-icon--failed' /> {this.props.error.message}
                </h3>
                <p>
                    <i className='fa fa-redo' /> <a href='javascript:document.location.reload();'>Reload this page</a> to try again.
                </p>
                {this.props.error.response && (
                    <>
                        {this.props.error.response.req && (
                            <>
                                <h5>Request</h5>
                                <pre>
                                    {this.props.error.response.req.method} {this.props.error.response.req.url}
                                </pre>
                            </>
                        )}
                        <>
                            <h5>Response</h5>
                            <pre>HTTP {this.props.error.response.status}</pre>
                            {this.props.error.response.body && <pre>{JSON.stringify(this.props.error.response.body, null, 2)}</pre>}
                        </>
                    </>
                )}
                <h5>Stack Trace</h5>
                <pre>{this.props.error.stack}</pre>
                {this.props.errorInfo && (
                    <>
                        <h5>Component Stack</h5>
                        <pre>{this.props.errorInfo.componentStack}</pre>
                    </>
                )}
            </div>
        );
    }
}

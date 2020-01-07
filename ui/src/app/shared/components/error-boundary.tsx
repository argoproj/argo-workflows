import {ErrorInfo} from 'react';
import * as React from 'react';
import {Page} from '../../../../node_modules/argo-ui';

interface State {
    error?: Error & {response?: any};
    errorInfo?: ErrorInfo;
}

export class ErrorBoundary extends React.Component<any, State> {
    public static getDerivedStateFromError(error: Error) {
        return {error};
    }

    constructor(props: any) {
        super(props);
        this.state = {};
    }

    public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
        this.setState({error, errorInfo});
    }

    public render() {
        if (this.state.error !== undefined) {
            return (
                <Page title='Error' toolbar={{}}>
                    <div className='argo-container'>
                        <div className=' white-box'>
                            <h3>
                                <i className='fa fa-skull fa-spin status-icon--failed' /> {this.state.error.message}
                            </h3>
                            <p>
                                <i className='fa fa-info-circle' /> Reload this page to dismiss this error message.
                            </p>
                            {this.state.error.response && (
                                <>
                                    {this.state.error.response.req && (
                                        <>
                                            <h5>Request</h5>
                                            <pre>
                                                {this.state.error.response.req.method} {this.state.error.response.req.url}
                                            </pre>
                                        </>
                                    )}
                                    <>
                                        <h5>Response</h5>
                                        <pre>HTTP {this.state.error.response.status}</pre>
                                        {this.state.error.response.body && <pre>{JSON.stringify(this.state.error.response.body, null, 2)}</pre>}
                                    </>
                                </>
                            )}
                            <h5>Stack Trace</h5>
                            <pre>{this.state.error.stack}</pre>
                            {this.state.errorInfo && (
                                <>
                                    <h5>Component Stack</h5>
                                    <pre>{this.state.errorInfo.componentStack}</pre>
                                </>
                            )}
                        </div>
                    </div>
                </Page>
            );
        }

        return this.props.children;
    }
}

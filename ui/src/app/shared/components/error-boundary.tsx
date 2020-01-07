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
                                <i className='fa fa-times-circle status-icon--failed' /> {this.state.error.message}
                            </h3>
                            <h5>Document</h5>
                            <p>{document.location.href}</p>
                            <>
                                {this.state.error.response.req && (
                                    <>
                                        <h5>Request</h5>
                                        <pre>
                                            {this.state.error.response.req.method} {this.state.error.response.req.url}
                                        </pre>
                                    </>
                                )}
                                {this.state.error.response.body && (
                                    <>
                                        <h5>Response</h5>
                                        <pre>{JSON.stringify(this.state.error.response.body, null, 2)}</pre>
                                    </>
                                )}
                            </>
                            <h5>Stack Trace</h5>
                            <pre>{this.state.error.stack}</pre>
                            {this.state.errorInfo && (
                                <>
                                    <h5>Component Stack</h5>
                                    <pre>{this.state.errorInfo.componentStack}</pre>
                                </>
                            )}
                            <p>
                                <i className='fa fa-info-circle' /> Reload this page to dismiss this error.
                            </p>
                        </div>
                    </div>
                </Page>
            );
        }

        return this.props.children;
    }
}

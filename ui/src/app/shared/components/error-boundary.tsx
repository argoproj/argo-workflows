import {ErrorInfo} from 'react';
import * as React from 'react';
import {Page} from '../../../../node_modules/argo-ui';

interface State {
    error?: Error;
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

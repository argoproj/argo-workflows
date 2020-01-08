import {ErrorInfo} from 'react';
import * as React from 'react';
import {ErrorPanel} from './error-panel';

interface State {
    error?: Error & {response?: any};
    errorInfo?: ErrorInfo;
}

export default class ErrorBoundary extends React.Component<any, State> {
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
                <div className='argo-container'>
                    <ErrorPanel {...this.state} />
                </div>
            );
        }

        return this.props.children;
    }
}

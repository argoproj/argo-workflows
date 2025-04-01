import {ErrorInfo} from 'react';
import * as React from 'react';

import {ErrorPanel} from './error-panel';

interface State {
    error?: Error & {response?: any};
    errorInfo?: ErrorInfo;
    hasError: boolean;
}

export default class ErrorBoundary extends React.Component<any, State> {
    public static getDerivedStateFromError(error: Error) {
        return {error, hasError: true};
    }

    constructor(props: any) {
        super(props);
        this.state = {
            hasError: false
        };
    }

    public componentDidCatch(error: Error, errorInfo: ErrorInfo) {
        // Log the error to console for debugging
        console.error('Error caught by ErrorBoundary:', error, errorInfo);
        this.setState({error, errorInfo, hasError: true});
    }

    public componentDidUpdate(prevProps: any) {
        // Reset error state when children change
        // This helps recover from errors when navigation occurs
        if (prevProps.children !== this.props.children && this.state.hasError) {
            this.setState({
                error: undefined,
                errorInfo: undefined,
                hasError: false
            });
        }
    }

    private handleRetry = () => {
        // Reset error state and retry rendering
        this.setState({
            error: undefined,
            errorInfo: undefined,
            hasError: false
        });
    };

    public render() {
        if (this.state.hasError) {
            return (
                <div className='argo-container'>
                    <div style={{marginBottom: '1em'}}>
                        <button className='argo-button argo-button--base' onClick={this.handleRetry}>
                            <i className='fa fa-redo'></i> Retry
                        </button>
                        <button className='argo-button argo-button--base-o' onClick={() => window.location.reload()} style={{marginLeft: '0.5em'}}>
                            <i className='fa fa-sync'></i> Reload Page
                        </button>
                    </div>
                    <ErrorPanel {...this.state} />
                </div>
            );
        }

        return this.props.children;
    }
}

import {ErrorInfo} from 'react';
import * as React from 'react';

interface State {
    error?: Error & {response?: any};
    errorInfo?: ErrorInfo;
    isReloading: boolean;
}

/**
 * Error boundary specifically for handling chunk loading failures.
 * Automatically reloads the page when a chunk fails to load.
 * Fixes: https://github.com/argoproj/argo-workflows/issues/15640
 */
export class ChunkLoadErrorBoundary extends React.Component<any, State> {
    static isChunkLoadError(error: Error): boolean {
        return (
            error.message.includes('Loading chunk') ||
            error.message.includes('Failed to fetch') ||
            error.message.includes('Failed to import') ||
            error.message.includes('NetworkError') ||
            error.name === 'ChunkLoadError'
        );
    }

    static getDerivedStateFromError(error: Error) {
        // Only handle chunk load errors; re-throw others
        if (ChunkLoadErrorBoundary.isChunkLoadError(error)) {
            return {error, isReloading: true};
        }
        throw error;
    }

    constructor(props: any) {
        super(props);
        this.state = {isReloading: false};
    }

    componentDidCatch(error: Error, errorInfo: ErrorInfo) {
        console.error('Chunk load error:', error, errorInfo);
        // Auto-reload after a brief delay to allow state update
        setTimeout(() => window.location.reload(), 100);
    }

    render() {
        if (this.state.isReloading) {
            return (
                <div style={{padding: '20px', textAlign: 'center'}}>
                    <h2>Reloading...</h2>
                    <p>A required component failed to load. The page will reload automatically.</p>
                </div>
            );
        }

        return this.props.children;
    }
}

import {ErrorInfo} from 'react';
import * as React from 'react';

import {ErrorPanel} from './error-panel';

interface State {
    error?: Error & {response?: any};
    errorInfo?: ErrorInfo;
}

const CHUNK_RELOAD_KEY = 'argo-chunk-reload-attempted';

function isChunkLoadError(error: Error): boolean {
    return error?.name === 'ChunkLoadError' || /Loading chunk \S+ failed/.test(error?.message ?? '');
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
        // After a redeploy, an open tab still references chunk hashes from the previous build,
        // so lazy import() 404s. Reload once to fetch the new index.html and current chunk manifest.
        if (isChunkLoadError(error) && !sessionStorage.getItem(CHUNK_RELOAD_KEY)) {
            sessionStorage.setItem(CHUNK_RELOAD_KEY, '1');
            window.location.reload();
            return;
        }
        sessionStorage.removeItem(CHUNK_RELOAD_KEY);
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

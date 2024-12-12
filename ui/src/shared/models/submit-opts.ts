export interface SubmitOpts {
    entryPoint?: string;
    parameters?: string[];
    labels?: string;
}

export interface ResubmitOpts {
    parameters?: string[];
    memoized?: boolean;
}

export interface RetryOpts {
    parameters?: string[];
    restartSuccessful?: boolean;
    nodeFieldSelector?: string;
}

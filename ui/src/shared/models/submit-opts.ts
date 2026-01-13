export interface SubmitOpts {
    entryPoint?: string;
    parameters?: string[];
    labels?: string;
    // Artifacts to override for the workflow
    // Format: name=s3://bucket/key or name=gcs://bucket/key
    artifacts?: string[];
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

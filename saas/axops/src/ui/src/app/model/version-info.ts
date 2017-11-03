export interface VersionInfo {
    namespace: string;
    version: string;
    cluster_id: string;
    features_set: 'full' | 'aws_limited' | 'limited' | 'lite';
}

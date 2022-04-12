import {Artifact, ArtifactRepository, NodeStatus, WorkflowStatus} from '../../models';

export const nodeArtifacts = (node: NodeStatus) =>
    (node.inputs?.artifacts || [])
        .map(a => ({
            ...a,
            artifactDiscriminator: 'input'
        }))
        .concat((node.outputs?.artifacts || []).map(a => ({...a, artifactDiscriminator: 'output'})));

export const artifactDescription = <A extends Artifact>(a: A, ar: ArtifactRepository) => {
    let urn = 'unknown';
    let desc = 'unknown';
    if (a.gcs) {
        desc = a.gcs.key;
        urn = 'artifact:gcs:' + (a.gcs.endpoint || ar.gcs?.endpoint) + ':' + (a.gcs.bucket || ar.gcs?.bucket) + ':' + desc;
    }
    if (a.git) {
        const revision = a.git.revision || 'HEAD';
        desc = a.git.repo + '#' + revision;
        urn = 'artifact:git:' + a.git.repo + ':' + revision;
    }
    if (a.http) {
        desc = a.http.url;
        urn = 'artifact:http::' + a.http.url;
    }
    if (a.s3) {
        desc = a.s3.key;
        urn = 'artifact:s3:' + (a.s3.endpoint || ar.s3.endpoint) + ':' + (a.s3?.bucket || ar.s3?.bucket) + ':' + desc;
    }
    if (a.oss) {
        desc = a.oss.key;
        urn = 'artifact:oss:' + (a.oss.endpoint || ar.oss?.endpoint) + ':' + (a.oss.bucket || ar.oss?.bucket) + ':' + desc;
    }
    if (a.raw) {
        desc = 'raw';
        urn = 'artifact:raw:' + a.raw.data;
    }
    return {
        ...a,
        urn,
        desc
    };
};

export const findArtifact = (status: WorkflowStatus, urn: string) => {
    const artifacts: (Artifact & {nodeId: string; artifactDiscriminator: string})[] = [];

    Object.values(status.nodes).map(node => {
        return nodeArtifacts(node)
            .map(a => ({
                ...a,
                ...artifactDescription(a, status.artifactRepositoryRef?.artifactRepository),
                nodeId: node.id
            }))
            .filter(ad => ad.urn === urn)
            .forEach(ad => artifacts.push(ad));
    });

    if (artifacts.length === 0) {
        return;
    }

    // return the last one
    return artifacts[artifacts.length - 1];
};

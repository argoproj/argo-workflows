import requests from './requests';

export interface ArtifactDescription {
    key?: string;
    contentType?: string;
    items?: {
        contentType?: string;
        name: string;
        size: number;
    }[];
}

export class ArtifactService {
    public getArtifactDescription(namespace: string, workflowName: string, nodeId: string, artifactDiscriminator: string, artifactName: string) {
        return requests
            .get(`artifact-descriptions/${namespace}/name/${workflowName}/${nodeId}/${artifactDiscriminator}/${artifactName}`)
            .then(res => res.body as ArtifactDescription);
    }
}

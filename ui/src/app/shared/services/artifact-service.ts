import requests from './requests';

export interface ArtifactDescription {
    filename?: string;
    contentType?: string;
    items?: {
        filename: string;
        size: number;
        contentType?: string;
    }[];
}

export class ArtifactService {
    public getArtifactDescription(namespace: string, workflowName: string, nodeId: string, artifactDiscrim: string, artifactName: string) {
        return requests
            .get(`workflow-artifacts/v2/artifact-descriptions/${namespace}/name/${workflowName}/${nodeId}/${artifactDiscrim}/${artifactName}`)
            .then(res => res.body as ArtifactDescription);
    }
}

import * as kubernetes from 'argo-ui/src/models/kubernetes';

export interface Pod {
    metadata: kubernetes.ObjectMeta;
    spec?: object;
    status?: object;
}

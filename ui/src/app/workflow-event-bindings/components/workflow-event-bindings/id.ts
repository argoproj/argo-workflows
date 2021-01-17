type Type = 'ClusterWorkflowTemplate' | 'WorkflowEventBinding' | 'WorkflowTemplate';

export const ID = {
    join: (type: Type, namespace: string, name: string) => type + '/' + namespace + '/' + name,
    split: (id: string) => ({
        type: id.split('/')[0] as Type,
        namespace: id.split('/')[1],
        name: id.split('/')[2]
    })
};

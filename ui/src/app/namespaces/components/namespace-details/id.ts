export const ID = {
    join: (x: {type: string; namespace: string; name: string; key?: string}) => x.namespace + '/' + x.type + '/' + x.name + (x.key ? '#' + x.key : ''),
    split: (id: string) => {
        const parts = id.split('/');
        const namespace = parts[0];
        const type = parts[1];
        const name = parts[2].split('#')[0];
        return {type, namespace, name};
    }
};

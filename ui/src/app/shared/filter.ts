import {ObjectMeta} from 'argo-ui/src/models/kubernetes';

export const searchToMetadataFilter = (search: string) => {
    const filters: ((md: ObjectMeta) => boolean)[] = [];
    search.split(' ').forEach(w => {
        if (w.startsWith('namespace:')) {
            filters.push((md: ObjectMeta) => md.namespace.indexOf(w.substring(10)) >= 0);
        } else if (w.startsWith('name:')) {
            filters.push((md: ObjectMeta) => md.name.indexOf(w.substring(5)) >= 0);
        } else {
            filters.push((md: ObjectMeta) => md.name.indexOf(w) >= 0 || md.namespace.indexOf(w) >= 0);
        }
    });
    return (md: ObjectMeta) => filters.map(f => f(md)).reduce((a, b) => a && b);
};

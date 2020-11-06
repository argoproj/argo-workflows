/**
 * Examples
 *
 * * argo-events/EventSource/webhook
 * * argo-events/EventType/webhook/example
 * * argo-events/Sensor/webhook
 * * argo-events/Trigger/webhook/example
 * * argo-events/Conditions/webhook/example - condition
 */

export const ID = {
    join: (x: {type: string; namespace: string; name: string; key?: string}) => x.namespace + '/' + x.type + '/' + x.name + (x.key ? '/' + x.key : ''),
    split: (id: string) => {
        const parts = id.split('/');
        const namespace = parts[0];
        const type = parts[1];
        const name = parts[2];
        const key = parts.length > 3 ? parts[3] : null;
        return {type, namespace, name, key};
    }
};

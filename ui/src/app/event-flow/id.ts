/**
 * Examples
 *
 * * argo-events/EventSource/webhook
 * * argo-events/EventSource/webhook/example
 * * argo-events/Sensor/webhook
 * * argo-events/Trigger/webhook/example
 * * argo-events/Conditions/webhook/example - condition
 * * argo-events/Workflow/webhook - condition
 */
type Type = 'EventSource' | 'Sensor' | 'Trigger' | 'Conditions' | 'Workflow' | 'Collapsed';

export const ID = {
    join: (type: Type, namespace: string, name: string, key?: string) => namespace + '/' + type + '/' + name + (key ? '/' + key : ''),
    split: (id: string) => {
        const parts = id.split('/');
        const namespace = parts[0];
        const type: Type = parts[1] as Type;
        const name = parts[2];
        const key = parts.length > 3 ? parts[3] : null;
        return {type, namespace, name, key};
    }
};

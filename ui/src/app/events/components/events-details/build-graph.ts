import {Condition, Workflow} from '../../../../models';
import {EventSource, eventSourceTypes} from '../../../../models/event-source';
import {Sensor, triggerTypes} from '../../../../models/sensor';
import {Graph, Node} from '../../../shared/components/graph/types';
import {icons} from './icons';
import {ID} from './id';

const status = (r: {status?: {conditions?: Condition[]}}) => {
    if (!r.status || !r.status.conditions) {
        return '';
    }
    return !!r.status.conditions.find(c => c.status !== 'True') ? 'Pending' : 'Ready';
};

export const buildGraph = (eventSources: EventSource[], sensors: Sensor[], workflows: Workflow[], flow: {[id: string]: any}) => {
    const edgeClassNames = (id: Node) => (!!flow[id] ? 'flow' : '');
    const graph = new Graph();

    (eventSources || []).forEach(eventSource => {
        Object.entries(eventSource.spec)
            .filter(([typeKey]) => ['template', 'service'].indexOf(typeKey) < 0)
            .forEach(([typeKey, type]) => {
                Object.keys(type).forEach(key => {
                    const eventId = ID.join('EventSource', eventSource.metadata.namespace, eventSource.metadata.name, key);
                    graph.nodes.set(eventId, {type: typeKey, label: key, classNames: status(eventSource), icon: icons[eventSourceTypes[typeKey] + 'EventSource']});
                });
            });
    });

    (sensors || []).forEach(sensor => {
        const sensorId = ID.join('Sensor', sensor.metadata.namespace, sensor.metadata.name);
        graph.nodes.set(sensorId, {type: 'sensor', label: sensor.metadata.name, icon: icons.Sensor, classNames: status(sensor)});
        (sensor.spec.dependencies || []).forEach(d => {
            const eventId = ID.join('EventSource', sensor.metadata.namespace, d.eventSourceName, d.eventName);
            graph.edges.set({v: eventId, w: sensorId}, {label: d.name, classNames: edgeClassNames(eventId)});
        });
        (sensor.spec.triggers || [])
            .map(t => t.template)
            .filter(template => template)
            .forEach(template => {
                const triggerTypeKey = Object.keys(template).filter(t => ['name', 'conditions'].indexOf(t) === -1)[0];
                const triggerId = ID.join('Trigger', sensor.metadata.namespace, sensor.metadata.name, template.name);
                graph.nodes.set(triggerId, {
                    label: template.name,
                    type: triggerTypeKey,
                    classNames: status(sensor),
                    icon: icons[triggerTypes[triggerTypeKey] + 'Trigger']
                });
                if (template.conditions) {
                    const conditionsId = ID.join('Conditions', sensor.metadata.namespace, sensor.metadata.name, template.conditions);
                    graph.nodes.set(conditionsId, {
                        type: 'conditions',
                        label: template.conditions,
                        icon: icons.Conditions,
                        classNames: ''
                    });
                    graph.edges.set({v: sensorId, w: conditionsId}, {classNames: edgeClassNames(sensorId)});
                    graph.edges.set({v: conditionsId, w: triggerId}, {classNames: edgeClassNames(triggerId)});
                } else {
                    graph.edges.set({v: sensorId, w: triggerId}, {classNames: edgeClassNames(triggerId)});
                }
            });
    });

    (workflows || []).forEach(workflow => {
        const sensorName = workflow.metadata.labels['events.argoproj.io/sensor'];
        const triggerName = workflow.metadata.labels['events.argoproj.io/trigger'];
        const phase = workflow.metadata.labels['workflows.argoproj.io/phase'];
        if (sensorName && triggerName) {
            const workflowId = ID.join('Workflow', workflow.metadata.namespace, workflow.metadata.name);
            graph.nodes.set(workflowId, {label: workflow.metadata.name, type: 'workflow', icon: icons.Workflow, classNames: phase});
            const triggerId = ID.join('Trigger', workflow.metadata.namespace, sensorName, triggerName);
            graph.edges.set({v: triggerId, w: workflowId}, {});
        }
    });

    return graph;
};

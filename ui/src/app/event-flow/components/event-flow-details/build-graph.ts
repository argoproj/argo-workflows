import {Condition, Workflow} from '../../../../models';
import {EventSource, EventSourceType} from '../../../../models/event-source';
import {Sensor, TriggerType} from '../../../../models/sensor';
import {Graph, Node} from '../../../shared/components/graph/types';
import {icons as phaseIcons} from '../../../workflows/components/workflow-dag/icons';
import {icons} from './icons';
import {ID} from './id';

function status(r: {status?: {conditions?: Condition[]}}) {
    if (!r.status || !r.status.conditions) {
        return '';
    }
    if (r.status.conditions.find(c => c.status === 'False')) {
        return 'Failed';
    }
    return r.status.conditions.find(c => c.status !== 'True') ? 'Pending' : 'Ready';
}

const numNodesToHide = 2;
export function buildGraph(eventSources: EventSource[], sensors: Sensor[], workflows: Workflow[], flow: {[p: string]: {count: number; timeout?: any}}, expanded: boolean) {
    const edgeLabel = (id: Node, label?: string) => (flow[id] ? (label || '') + ' (' + flow[id].count + ')' : label);
    const edgeClassNames = (id: Node) => (!!flow[id] && flow[id].timeout ? 'flow' : '');
    const graph = new Graph();

    (eventSources || []).forEach(eventSource => {
        Object.entries(eventSource.spec)
            .filter(([typeKey]) => ['template', 'service'].indexOf(typeKey) < 0)
            .forEach(([typeKey, type]) => {
                Object.keys(type).forEach(key => {
                    const eventId = ID.join('EventSource', eventSource.metadata.namespace, eventSource.metadata.name, key);
                    graph.nodes.set(eventId, {genre: typeKey as EventSourceType, label: key, classNames: status(eventSource), icon: icons[typeKey + 'EventSource']});
                });
            });
    });

    (sensors || []).forEach(sensor => {
        const sensorId = ID.join('Sensor', sensor.metadata.namespace, sensor.metadata.name);
        graph.nodes.set(sensorId, {genre: 'sensor', label: sensor.metadata.name, icon: icons.sensor, classNames: status(sensor)});
        (sensor.spec.dependencies || []).forEach(d => {
            const eventId = ID.join('EventSource', sensor.metadata.namespace, d.eventSourceName, d.eventName);
            graph.edges.set({v: eventId, w: sensorId}, {label: edgeLabel(eventId, d.name), classNames: edgeClassNames(eventId)});
        });
        (sensor.spec.triggers || [])
            .map(t => t.template)
            .filter(template => template)
            .forEach(template => {
                const triggerTypeKey = Object.keys(template).filter(t => ['name', 'conditions'].indexOf(t) === -1)[0];
                const triggerId = ID.join('Trigger', sensor.metadata.namespace, sensor.metadata.name, template.name);
                graph.nodes.set(triggerId, {
                    label: template.name,
                    genre: triggerTypeKey as TriggerType,
                    classNames: status(sensor),
                    icon: icons[triggerTypeKey + 'Trigger']
                });
                if (template.conditions) {
                    const conditionsId = ID.join('Conditions', sensor.metadata.namespace, sensor.metadata.name, template.conditions);
                    graph.nodes.set(conditionsId, {
                        genre: 'conditions',
                        label: template.conditions,
                        icon: icons.conditions,
                        classNames: ''
                    });
                    graph.edges.set({v: sensorId, w: conditionsId}, {label: edgeLabel(sensorId), classNames: edgeClassNames(sensorId)});
                    graph.edges.set({v: conditionsId, w: triggerId}, {label: edgeLabel(triggerId), classNames: edgeClassNames(triggerId)});
                } else {
                    graph.edges.set({v: sensorId, w: triggerId}, {label: edgeLabel(triggerId), classNames: edgeClassNames(triggerId)});
                }
            });
    });

    const workflowGroups: {[triggerId: string]: Workflow[]} = {};

    (workflows || []).forEach(workflow => {
        const sensorName = workflow.metadata.labels['events.argoproj.io/sensor'];
        const triggerName = workflow.metadata.labels['events.argoproj.io/trigger'];
        const triggerId = ID.join('Trigger', workflow.metadata.namespace, sensorName, triggerName);
        if (!workflowGroups[triggerId]) {
            workflowGroups[triggerId] = [];
        }
        workflowGroups[triggerId].push(workflow);
    });

    Object.entries(workflowGroups).forEach(([triggerId, items]) => {
        items.forEach((workflow, i) => {
            // we always add workflows if:
            // 1. We are showing expanded view.
            // 2. The workflow is amongst the first 2 in the list.
            // 3. We're showing <= 3 workflows.
            if (expanded || i < numNodesToHide || items.length <= numNodesToHide + 1) {
                const workflowId = ID.join('Workflow', workflow.metadata.namespace, workflow.metadata.name);
                const phase = workflow.metadata.labels['workflows.argoproj.io/phase'];
                graph.nodes.set(workflowId, {
                    label: workflow.metadata.name,
                    genre: 'workflow',
                    icon: phaseIcons[phase] || phaseIcons.Pending,
                    classNames: phase
                });
                graph.edges.set({v: triggerId, w: workflowId}, {classNames: edgeClassNames(workflowId)});
            } else if (i === numNodesToHide) {
                // use "3" to make sure we only add it once
                const workflowGroupId = ID.join('Collapsed', workflow.metadata.namespace, triggerId);
                graph.nodes.set(workflowGroupId, {
                    label: items.length - numNodesToHide + ' hidden workflow(s)',
                    genre: 'collapsed',
                    icon: icons.collapsed
                });
                graph.edges.set({v: triggerId, w: workflowGroupId}, {classNames: ''});
            }
        });
    });

    return graph;
}

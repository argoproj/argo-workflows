import {Pipeline} from '../../../../models/pipeline';
import {Step} from '../../../../models/step';
import {Graph} from '../../../shared/components/graph/types';
import {Icon} from '../../../shared/components/icon';

type Type = '' | 'cat' | 'container' | 'filter' | 'git' | 'group' | 'handler' | 'map';

const stepIcon = (type: Type): Icon => {
    switch (type) {
        case 'cat':
            return 'exchange-alt';
        case 'container':
            return 'cube';
        case 'filter':
            return 'filter';
        case 'git':
            return 'code-branch';
        case 'group':
            return 'object-group';
        case 'handler':
            return 'code';
        case 'map':
            return 'exchange-alt';
        default:
            return 'square';
    }
};

const topicIcon: Icon = 'inbox';
const pendingSymbol = '🕑';
const errorSymbol = '⚠️ ';
const totalSymbol = 'x';

const emptySet = '∅';
export const graph = (pipeline: Pipeline, steps: Step[]) => {
    const g = new Graph();

    steps.forEach(step => {
        const spec = step.spec;
        const stepId = 'step/' + spec.name;
        const status = step.status || {phase: '', replicas: 0};

        const type: Type = spec.cat
            ? 'cat'
            : spec.container
            ? 'container'
            : spec.filter
            ? 'filter'
            : spec.git
            ? 'git'
            : spec.group
            ? 'group'
            : spec.handler
            ? 'handler'
            : spec.map
            ? 'map'
            : '';

        const nodeLabel = status.replicas !== 1 ? spec.name + ' (x' + status.replicas + ')' : spec.name;
        g.nodes.set(stepId, {genre: type, label: nodeLabel, icon: stepIcon(type), classNames: status.phase});

        const classNames = status.phase === 'Running' ? 'flow' : '';
        (spec.sources || []).forEach((x, i) => {
            const ss = (status.sourceStatuses || {})[x.name || ''] || {};
            const metrics = Object.values(ss.metrics || {}).reduce(
                (a, b) => ({
                    total: (a.total || 0) + (b.total || 0),
                    errors: (a.errors || 0) + (b.errors || 0)
                }),
                {total: 0, errors: 0}
            );
            const label =
                (metrics.errors > 0 ? errorSymbol + metrics.errors : '') +
                (ss.pending ? pendingSymbol + ss.pending : '') +
                ' ' +
                totalSymbol +
                (metrics.total || '?') +
                ' (' +
                ((ss.lastMessage || {}).data || emptySet) +
                ')';
            if (x.cron) {
                const cronId = 'cron/' + stepId + '/' + x.cron.schedule;
                g.nodes.set(cronId, {genre: 'cron', icon: 'clock', label: x.cron.schedule});
                g.edges.set({v: cronId, w: stepId}, {classNames, label});
            } else if (x.kafka) {
                const kafkaId = x.kafka.name || x.kafka.url || 'default';
                const topicId = 'kafka/' + kafkaId + '/' + x.kafka.topic;
                g.nodes.set(topicId, {genre: 'kafka', icon: topicIcon, label: x.kafka.topic});
                g.edges.set({v: topicId, w: stepId}, {classNames, label});
            } else if (x.stan) {
                const stanId = x.stan.name || x.stan.url || 'default';
                const subjectId = 'stan/' + stanId + '/' + x.stan.subject;
                g.nodes.set(subjectId, {genre: 'stan', icon: topicIcon, label: x.stan.subject});
                g.edges.set({v: subjectId, w: stepId}, {classNames, label});
            }
        });
        (spec.sinks || []).forEach((x, i) => {
            const ss = (status.sinkStatuses || {})[x.name || ''] || {};
            const metrics = Object.values(ss.metrics || {}).reduce(
                (a, b) => ({
                    total: (a.total || 0) + (b.total || 0),
                    errors: (a.errors || 0) + (b.errors || 0)
                }),
                {total: 0, errors: 0}
            );
            const label =
                (metrics.errors > 0 ? errorSymbol + metrics.errors : '') +
                (ss.pending ? pendingSymbol + ss.pending : '') +
                ' ' +
                totalSymbol +
                (metrics.total || '?') +
                ' (' +
                ((ss.lastMessage || {}).data || emptySet) +
                ')';
            if (x.kafka) {
                const kafkaId = x.kafka.name || x.kafka.url || 'default';
                const topicId = 'kafka/' + kafkaId + '/' + x.kafka.topic;
                g.nodes.set(topicId, {genre: 'kafka', icon: topicIcon, label: x.kafka.topic});
                g.edges.set({v: stepId, w: topicId}, {classNames, label});
            } else if (x.log) {
                const logId = 'log/' + stepId;
                g.nodes.set(logId, {genre: 'log', icon: 'file-alt', label: 'log'});
                g.edges.set({v: stepId, w: logId}, {classNames, label});
            } else if (x.stan) {
                const stanId = x.stan.name || x.stan.url || 'default';
                const subjectId = 'stan/' + stanId + '/' + x.stan.subject;
                g.nodes.set(subjectId, {genre: 'stan', icon: topicIcon, label: x.stan.subject});
                g.edges.set({v: stepId, w: subjectId}, {classNames, label});
            }
        });
    });
    return g;
};

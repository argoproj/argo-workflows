import {Pipeline} from '../../../../models/pipeline';
import {Metrics, Step} from '../../../../models/step';
import {Graph} from '../../../shared/components/graph/types';
import {Icon} from '../../../shared/components/icon';
import {parseResourceQuantity} from '../../../shared/resource-quantity';
import {recent} from './recent';

type Type = '' | 'cat' | 'container' | 'dedupe' | 'expand' | 'filter' | 'flatten' | 'git' | 'group' | 'handler' | 'map';

const stepIcon = (type: Type): Icon => {
    switch (type) {
        case 'cat':
            return 'arrows-alt-h';
        case 'container':
            return 'cube';
        case 'dedupe':
            return 'filter';
        case 'expand':
            return 'expand';
        case 'filter':
            return 'filter';
        case 'flatten':
            return 'compress';
        case 'git':
            return 'code-branch';
        case 'group':
            return 'object-group';
        case 'handler':
            return 'code';
        case 'map':
            return 'arrows-alt-h';
        default:
            return 'square';
    }
};

const pendingSymbol = '🕑';
const errorSymbol = '⚠️';

const formatRates = (metrics: Metrics, replicas: number) => {
    const rates = Object.entries(metrics || {})
        // the rate will remain after scale-down, so we must filter out, as it'll be wrong
        .filter(([replica, m]) => parseInt(replica, 10) < replicas);
    return rates.length > 0
        ? '＊' +
              rates
                  .map(([, m]) => m)
                  .map(m => parseResourceQuantity(m.rate))
                  .reduce((a, b) => a + b, 0)
                  .toPrecision(3)
        : '';
};

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
            : spec.dedupe
            ? 'dedupe'
            : spec.expand
            ? 'expand'
            : spec.filter
            ? 'filter'
            : spec.git
            ? 'git'
            : spec.flatten
            ? 'flatten'
            : spec.group
            ? 'group'
            : spec.handler
            ? 'handler'
            : spec.map
            ? 'map'
            : '';

        const nodeLabel = status.replicas !== 1 ? spec.name + ' (x' + (status.replicas || 0) + ')' : spec.name;
        g.nodes.set(stepId, {genre: type, label: nodeLabel, icon: stepIcon(type), classNames: status.phase});

        const classNames = status.phase === 'Running' ? 'flow' : '';
        (spec.sources || []).forEach((x, i) => {
            const ss = (status.sourceStatuses || {})[x.name || ''] || {};

            const label =
                (recent(ss.lastError && new Date(ss.lastError.time)) ? errorSymbol : '') +
                (ss.pending ? ' ' + pendingSymbol + ss.pending + ' ' : '') +
                formatRates(ss.metrics, step.status.replicas);
            if (x.cron) {
                const cronId = 'cron/' + stepId + '/' + x.cron.schedule;
                g.nodes.set(cronId, {genre: 'cron', icon: 'clock', label: x.cron.schedule});
                g.edges.set({v: cronId, w: stepId}, {classNames, label});
            } else if (x.kafka) {
                const kafkaId = x.kafka.name || x.kafka.url || 'default';
                const topicId = 'kafka/' + kafkaId + '/' + x.kafka.topic;
                g.nodes.set(topicId, {genre: 'kafka', icon: 'inbox', label: x.kafka.topic});
                g.edges.set({v: topicId, w: stepId}, {classNames, label});
            } else if (x.stan) {
                const stanId = x.stan.name || x.stan.url || 'default';
                const subjectId = 'stan/' + stanId + '/' + x.stan.subject;
                g.nodes.set(subjectId, {genre: 'stan', icon: 'inbox', label: x.stan.subject});
                g.edges.set({v: subjectId, w: stepId}, {classNames, label});
            } else if (x.http) {
                const y = new URL('http://' + pipeline.metadata.name + '-' + step.spec.name + '/sources/' + x.name);
                const subjectId = 'http/' + y;
                g.nodes.set(subjectId, {genre: 'http', icon: 'cloud', label: y.hostname});
                g.edges.set({v: subjectId, w: stepId}, {classNames, label});
            }
        });
        (spec.sinks || []).forEach((x, i) => {
            const ss = (status.sinkStatuses || {})[x.name || ''] || {};
            const label = (recent(ss.lastError && new Date(ss.lastError.time)) ? errorSymbol : '') + formatRates(ss.metrics, step.status.replicas);
            if (x.kafka) {
                const kafkaId = x.kafka.name || x.kafka.url || 'default';
                const topicId = 'kafka/' + kafkaId + '/' + x.kafka.topic;
                g.nodes.set(topicId, {genre: 'kafka', icon: 'inbox', label: x.kafka.topic});
                g.edges.set({v: stepId, w: topicId}, {classNames, label});
            } else if (x.log) {
                const logId = 'log/' + stepId + '/' + x.name;
                g.nodes.set(logId, {genre: 'log', icon: 'file-alt', label: 'log'});
                g.edges.set({v: stepId, w: logId}, {classNames, label});
            } else if (x.stan) {
                const stanId = x.stan.name || x.stan.url || 'default';
                const subjectId = 'stan/' + stanId + '/' + x.stan.subject;
                g.nodes.set(subjectId, {genre: 'stan', icon: 'inbox', label: x.stan.subject});
                g.edges.set({v: stepId, w: subjectId}, {classNames, label});
            } else if (x.http) {
                const y = new URL(x.http.url);
                const subjectId = 'http/' + y;
                g.nodes.set(subjectId, {genre: 'http', icon: 'cloud', label: y.hostname});
                g.edges.set({v: stepId, w: subjectId}, {classNames, label});
            }
        });
    });
    return g;
};

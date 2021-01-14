import {Icon} from '../../../shared/components/icon';

const chat = 'comment';
const compute = 'microchip';
const git = 'code-branch';
const grid = 'th';
const manifest = 'file-code';
const queue = 'stream';
const storage = 'database';
const web = 'cloud';

export const icons: {[type: string]: Icon} = {
    amqpEventSource: queue,
    awsLambdaTrigger: compute,
    argoWorkflowTrigger: 'stream',
    azureEventsHubEventSource: storage,
    calendarEventSource: 'clock',
    collapsed: 'ellipsis-h',
    conditions: 'filter',
    customTrigger: 'puzzle-piece',
    emitterEventSource: queue,
    fileEventSource: 'file',
    genericEventSource: 'puzzle-piece',
    githubEventSource: git,
    gitlabEventSource: git,
    hdfsEventSource: 'hdd',
    httpTrigger: web,
    k8sTrigger: manifest,
    kafkaEventSource: queue,
    kafkaTrigger: queue,
    logTrigger: 'file-alt',
    minioEventSource: storage,
    mqttEventSource: queue,
    natsEventSource: queue,
    natsTrigger: queue,
    nsqEventSource: queue,
    openWhiskTrigger: compute,
    pubSubEventSource: queue,
    pulsarEventSource: queue,
    redisEventSource: grid,
    resourceEventSource: manifest,
    snsEventSource: queue,
    sqsEventSource: queue,
    sensor: 'satellite-dish', // resource type
    slackEventSource: chat,
    slackTrigger: chat,
    storageGridEventSource: grid,
    stripeEventSource: 'credit-card',
    webhookEventSource: web
};

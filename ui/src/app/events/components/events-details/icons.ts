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
    AMQPEventSource: queue,
    AWSLambdaTrigger: compute,
    ArgoWorkflowTrigger: 'sitemap',
    AzureEventsHubEventSource: storage,
    CalendarEventSource: 'clock',
    Conditions: 'filter', // special type
    CustomTrigger: 'puzzle-piece',
    EmitterEventSource: queue,
    FileEventSource: 'file',
    GenericEventSource: 'puzzle-piece',
    GithubEventSource: git,
    GitlabEventSource: git,
    HDFSEventSource: 'hdd',
    HTTPTrigger: web,
    K8STrigger: manifest,
    KafkaEventSource: queue,
    KafkaTrigger: queue,
    LogTrigger: 'file-alt',
    MinioEventSource: storage,
    MQTTEventSource: queue,
    NATSEventSource: queue,
    NATSTrigger: queue,
    NSQEventSource: queue,
    OpenWhiskTrigger: compute,
    PubSubEventSource: queue,
    PulsarEventSource: queue,
    RedisEventSource: grid,
    ResourceEventSource: manifest,
    SNSEventSource: queue,
    SQSEventSource: queue,
    Sensor: 'circle', // resource type
    SlackEventSource: chat,
    SlackTrigger: chat,
    StorageGridEventSource: grid,
    StripeEventSource: 'credit-card',
    WebhookEventSource: web,
    Workflow: 'sitemap'
};

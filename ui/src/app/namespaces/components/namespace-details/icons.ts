const chat = 'comment';
const compute = 'microchip';
const git = 'code-branch';
const grid = 'th';
const manifest = 'file-code';
const queue = 'stream';
const storage = 'database';
const web = 'cloud';

export const icons: {[type: string]: string} = {
    AMQPEventType: queue,
    AWSLambdaTrigger: compute,
    ArgoWorkflowTrigger: 'sitemap',
    AzureEventsHubEventType: storage,
    CalendarEventType: 'clock',
    Conditions: 'filter', // special type
    CustomTrigger: 'puzzle-piece',
    EmitterEventType: queue,
    FileEventType: 'file',
    GenericEventType: 'puzzle-piece',
    GithubEventType: git,
    GitlabEventType: git,
    HDFSEventType: 'hdd',
    HTTPTrigger: web,
    K8STrigger: manifest,
    KafkaEventType: queue,
    KafkaTrigger: queue,
    MinioEventType: storage,
    MQTTEventType: queue,
    NATSEventType: queue,
    NATSTrigger: queue,
    NSQEventType: queue,
    OpenWhiskTrigger: compute,
    PubSubEventType: queue,
    PulsarEventType: queue,
    RedisEventType: grid,
    ResourceEventType: manifest,
    SNSEventType: queue,
    SQSEventType: queue,
    Sensor: 'circle', // resource type
    SlackEventType: chat,
    SlackTrigger: chat,
    StorageGridEventType: grid,
    StripeEventType: 'credit-card',
    WebhookEventType: web
};

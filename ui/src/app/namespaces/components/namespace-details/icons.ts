const chat = 'comment';
const compute = 'microchip';
const git = 'code-branch';
const grid = 'th';
const manifest = 'file-code';
const queue = 'stream'; 
const storage = 'database';
const web = "cloud";

export const icons: { [type: string]: string } = {
    AMQPEvent: queue,
    AWSLambdaTrigger: compute,
    ArgoWorkflowTrigger: 'sitemap',
    AzureEventsHubEvent: storage,
    CalendarEvent: 'clock',
    Conditions: 'filter', // special type
    Dependency: 'link', // special type
    CustomTrigger: 'puzzle-piece',
    EmitterEvent: queue,
    Event: queue, // fall-back type
    EventSource: 'circle', // resource type
    FileEvent: 'file',
    GenericEvent: 'puzzle-piece',
    GithubEvent: git,
    GitlabEvent: git,
    HDFSEvent: 'hdd',
    HTTPTrigger: web,
    K8STrigger: manifest,
    KafkaEvent: queue,
    KafkaTrigger: queue,
    MinioEvent: storage,
    MQTTEvent: queue,
    NATSEvent: queue,
    NATSTrigger: queue,
    NSQEvent: queue,
    OpenWhiskTrigger: compute,
    PubSubEvent: queue,
    PulsarEvent: queue,
    RedisEvent: grid,
    ResourceEvent: manifest,
    SNSEvent: queue,
    SQSEvent: queue,
    Sensor: 'circle', // resource type
    SlackEvent: chat,
    SlackTrigger: chat,
    StorageGridEvent: grid,
    StripeEvent: 'credit-card',
    Trigger: 'bell', // fall-back type
    WebhookEvent: web
};
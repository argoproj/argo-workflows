export class Rule {
    rule_id: string;
    name: string;
    recipients: string[];
    channels: string[];
    severities: string[];
    enabled?: boolean;
    codes?: string[];
}

export class NotificationEvent {
    channel: string;
    cluster: string;
    code: string;
    detail: string[];
    event_id: string;
    facility: string;
    message: string;
    recipients: string[];
    severity: string;
    trace_id: string;
    timestamp: number;
    acknowledged_by?: string;
}

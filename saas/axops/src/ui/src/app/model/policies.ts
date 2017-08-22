export class Policy {
    public id: string;
    public name: string;
    public description: string;
    public repo: string;
    public branch: string;
    public author: string;
    public template: string;
    public enabled: boolean;
    public deleted: boolean;
    public parameters: {[key: string]: string};
    public notifications: PolicyNotification[];
    public when: PolicyEvent[];
    public selected: boolean;
    public status: string;
}

export class PolicyNotification {
    whom: string[];
    when: string[];
}

export class PolicyEvent {
    event: string;
    target_branches: string[];
    schedule: string;
    timezone: string;
}

export const DEFAULT_NOTIFICATIONS: PolicyNotification[] = [{
    whom: ['submitter'],
    when: ['on_success', 'on_failure']
}];

export const PolicyFieldNames = {
    name: 'name',
    description: 'description',
    repo: 'repo',
    branch: 'branch',
    subtype: 'subtype',
    cost: 'cost',
};

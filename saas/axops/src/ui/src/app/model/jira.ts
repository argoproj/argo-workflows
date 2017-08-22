export class JiraIssue {
    public key?: string;
    public issuetype: string;
    public project: string;
    public description: string;
    public summary: string;
    public reporter: string;
}

export const JIRA_ISSUE_TYPE = {
    'EPIC': '10000',
    'STORY': '10001',
    'TASK': '10002',
    'SUBTASK': '10003',
    'BUG': '10004',
};

export class JiraProject {
    public key: string;
    public id: string;
    public projectTypeKey: string;
    public name: string;
}

export class JiraTicket {
    assignee: JiraUser;
    comment: {
        comments: any[];
    };
    created: any;
    creator: JiraUser;
    description: string;
    id: string;
    issueType: {
        description: string;
        id: string;
        name: string;
        subtask: boolean;
    };
    key: string;
    labels: string[];
    priority: {
        id: string;
        name: string;
    };
    project: {
        id: string;
        key: string;
        name: string;
    };
    reporter: JiraUser;
    status: {
        description: string;
        id: string;
        name: string;
    };
    subtasks: any[];
    summary: string;
    updated: any;
    url: string;
    versions: any[];
    workratio: number;
}

export class JiraUser {
    accountId: string;
    active: boolean;
    displayName: string;
    emialAddress: string;
    key: string;
    name: string;
}

export class JiraIssueResponse {
    description: string;
    key: string;
    project: {
        id: string;
        key: string;
        name: string;
    };
    status: {
        id: string;
        name: string;
    };
    summary: string;
}

import { Commit } from './commit';
import { Template } from './template';
import { PolicyNotification } from './policies';

export type StaticFixtureInfo = { name: string, service_ids: { service_id: string, reference_name: string }[] } & { [ name: string]: any };

export interface TaskCreationArgs {
    template?: Template;
    template_id?: string;
    arguments: { [name: string]: string };
    user?: string;
    dry_run?: boolean;
    notifications?: PolicyNotification[];
}

export class Task {
    id: string = '';
    artifact_tags: string = '';
    name: string = '';
    app: string = '';
    desc: string = '';
    endpoint: string = '';
    user: string = '';
    stage: number = 0;
    ctime: number = 0;
    mtime: number = 0;
    wtime: number = 0;
    run_time: number = 0;
    average_runtime: number = 0;
    status: number = 0;
    container_id: string = '';
    log: string = '';
    subtasks: Task[] = [];
    template: Template = {};
    commit: Commit = new Commit();
    children: Task[];
    launch_time: number;
    init_time: number;
    wait_time: number;
    create_time: number;
    arguments: any = {};
    failure_path: string[] = [];
    labels: Object = {};
    requirements: Object = {};
    template_id: string = '';
    task_id: string = '';
    jira_issues?: string[] = [];
    fixtures?: { [name: string]: StaticFixtureInfo };
    cost: number;

    constructor(data?) {
        if (data && typeof data === 'object') {
            Object.assign(this, data);
        }
    }
}

export const TaskFieldNames = {
    name: 'name',
    app: 'app',
    desc: 'desc',
    description: 'description',
    endpoint: 'endpoint',
    username: 'username',
    user: 'user',
    stage: 'stage',
    status: 'status',
    status_string: 'status_string',
    containerId: 'container_id',
    log: 'log',
    subtasks: 'subtasks',
    template: 'template',
    commit: 'commit',
    children: 'children',
    failurePath: 'failure_path',
    labels: 'labels',
    parameters: 'parameters',
    templateId: 'template_id',
    repo: 'repo',
    branch: 'branch',
    jira_issues: 'jira_issues',
    policy_id: 'policy_id'
};

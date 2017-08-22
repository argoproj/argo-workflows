import { Template } from './template';

export interface FixtureAttribute {
    type: 'string' | 'bool' | 'int' | 'float';
    default?: string;
    flags?: string;
    options?: string[];
}

export class FixtureClass {
    action_templates: { [name: string]: Template };
    actions: { [name: string]: { parameters: any, template: string} };
    attributes: { [name: string]: FixtureAttribute };
    branch: string;
    description: string;
    id: string;
    name: string;
    repo: string;
    repo_branch: string;
    revision: string;
}

export interface FixtureTemplate {
    actions: { [name: string]: { parameters: any, template: string} };
    attributes: { [name: string]: FixtureAttribute };
    branch: string;
    description: string;
    id: string;
    name: string;
    repo: string;
    revision: string;
}

export const FixtureStatuses = {
    INIT: 'init',
    CREATING: 'creating',
    CREATE_ERROR: 'create_error',
    ACTIVE: 'active',
    OPERATING: 'operating',
    DELETING: 'deleting',
    DELETE_ERROR: 'delete_error',
    DELETED: 'deleted',
};

export const FixtureActions = {
    CREATE: 'create',
    DELETE: 'delete',
};

export class FixtureInstance {
    attributes?: any;
    class?: string;
    class_id?: string;
    class_name?: string;
    compatible?: boolean;
    concurrency?: number;
    creator?: string;
    description?: string;
    disable_reason?: string;
    enabled?: boolean;
    history?: any [];
    id?: string;
    name?: string;
    operation?: { id: string, name: string };
    owner?: string;
    referrers?: any [];
    status?: string;
    status_detail?: any;
}

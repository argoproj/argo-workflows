export type ToolCategory = 'scm' | 'notification' | 'registry' | 'authentication' | 'domain_management' | 'artifact_management' | 'registry' | 'issue_management';

export interface ITool {
    id?: string;
    type?: string;
    url?: string;
    nickname?: string;
    username?: string;
    password?: string;
    admin_address?: string;
    port?: number;
    timeout?: number;
    use_tls?: boolean;
    projects?: string[];
    all_repos?: string[];
    repos?: string[];
    use_webhook?: boolean;
    private_key?: string;
    public_cert?: string;
    hostname?: string;
    oauth_token?: string;
}

export class Tool {
    id: string = '';
    type: string = '';
    url: string = '';
    username: string = '';
    password: string = '';
    category: string = '';
    use_webhook: boolean = false;
}

export class CertificateTool {
    id: string = '';
    type: string = '';
    category: string = '';
    private_key: string = '';
    public_cert: string = '';
}

export class NotificationTool {
    id: string = '';
    type: string = '';
    url: string = '';
    nickname: string = '';
    username: string = '';
    password: string = '';
    admin_address: string = '';
    port: number;
    timeout: number;
    use_tls: boolean = false;

    constructor() {
        this.timeout = 10;
        this.use_tls = true;
        this.port = 587;
    }
}

export class SlackTool {
    id: string = '';
    type: string = '';
    oauth_token: string = '';
}

export class NexusTool {
    id: string = '';
    type: string = '';
    hostname: string = '';
    port: number;
    username: string;
    password: string;

    constructor() {
        this.username = '';
        this.password = '';
        this.hostname = '';
        this.port = 8081;
    }
}

export class JiraTool {
    id: string = '';
    type: string = '';
    hostname: string;
    username: string;
    password: string;
    category?: string;
    projects?: any;
    url?: string;

    constructor() {
        this.hostname = '';
        this.username = '';
        this.password = '';
    }
}

export class SamlConfigTool {
    id: string = '';
    category: string = '';
    type: string = '';
    button_label: string = '';
    deflate_response_encoded: boolean = false;
    email_attribute: string = 'User.Email';
    first_name_attribute: string = 'User.FirstName';
    group_attribute: string = 'User.Group';
    idp_public_cert: string = '';
    idp_sso_url: string = '';
    last_name_attribute: string = 'User.LastName';
    sign_request: boolean = true;
    signed_response: boolean = true;
    signed_response_assertion: boolean = false;
    sp_description: string = 'Argo Enterprise DevOps';
    sp_display_name: string = 'Argo';
    // We do not show url in UI
    url: string = '';
}

export interface SamlInfo {
    entity_id: string;
    sso_callback_url: string;
    public_cert: string;
}

export class ContainerRegistry {
    category: string = 'registry';
    type: string = '';
    password: string = '';
    username: string = '';
    hostname: string = '';
    id: string = null;

    constructor(type: string) {
        this.type = type;
        this.category = 'registry';
    }

}

export const REGISTRY_TYPES = {
    dockerhub: 'dockerhub',
    privateRegistry: 'private_registry'
};

export class Route53Config {
    id: string;
    url: string;
    category: string;
    type: string;
    domains: any[];
    all_domains: string[];

    constructor(data?) {
        if (typeof data === 'object') {
            for (let key in data) {
                if (data.hasOwnProperty(key)) {
                    this[key] = data[key];
                }
            }
        }
    }
}

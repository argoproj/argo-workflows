export interface Link {
    name: string;
    scope: string;
    url: string;
}

export interface Info {
    modals: {string: boolean};
    managedNamespace?: string;
    links?: Link[];
    navColor?: string;
}

export interface Version {
    version: string;
}

export interface GetUserInfoResponse {
    subject?: string;
    issuer?: string;
    groups?: string[];
    email?: string;
    emailVerified?: boolean;
    serviceAccountName?: string;
    serviceAccountNamespace?: string;
}

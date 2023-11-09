export interface Link {
    name: string;
    scope: string;
    url: string;
}

export interface Column {
    name: string;
    type: string;
    key: string;
}
export interface Info {
    modals: {string: boolean};
    managedNamespace?: string;
    links?: Link[];
    navColor?: string;
    columns: Column[];
}

export interface Version {
    version: string;
}

export interface GetUserInfoResponse {
    subject?: string;
    issuer?: string;
    groups?: string[];
    name?: string;
    email?: string;
    emailVerified?: boolean;
    serviceAccountName?: string;
    serviceAccountNamespace?: string;
}

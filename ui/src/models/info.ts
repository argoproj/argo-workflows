export interface Link {
    name: string;
    scope: string;
    url: string;
}

export interface UI {
    navColor?: string;
}

export interface Info {
    managedNamespace?: string;
    links?: Link[];
    ui?: UI;
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
}

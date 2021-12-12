export interface Link {
    name: string;
    scope: string;
    url: string;
}

export interface Ui {
    navColor?: string;
}

export interface Info {
    managedNamespace?: string;
    links?: Link[];
    ui?: Ui;
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

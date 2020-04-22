export interface Link {
    name: string;
    scope: string;
    url: string;
}

export interface Info {
    managedNamespace?: string;
    links?: Link[];
}

export interface Version {
    version: string;
}

export interface Link {
    name: string;
    scope: string;
    url: string;
}
export interface User {
    name: string;
}

export interface Info {
    managedNamespace?: string;
    links?: Link[];
    user?: User;
}

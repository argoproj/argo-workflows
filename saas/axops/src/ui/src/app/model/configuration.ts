export interface Configuration {
    description?: string;
    is_secret?: boolean;
    name?: string;
    user?: string;
    value?: {[name: string]: string};
    ctime?: number;
    mtime?: number;
}

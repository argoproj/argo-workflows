export interface Configuration {
    description?: string;
    name?: string;
    user?: string;
    value?: {[name: string]: string};
    ctime?: number;
    mtime?: number;
}

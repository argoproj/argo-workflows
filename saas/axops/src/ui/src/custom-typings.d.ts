declare var ENV: string;
declare var HMR: boolean;
declare var System: SystemJS;

interface SystemJS {
    import: (path?: string) => Promise<any>;
}

interface GlobalEnvironment {
    ENV: string;
    HMR: boolean;
    SystemJS: SystemJS;
    System: SystemJS;
}

interface Es6PromiseLoader {
    (id: string): (exportName?: string) => Promise<any>;
}

type FactoryEs6PromiseLoader = () => Es6PromiseLoader;
type FactoryPromise = () => Promise<any>;

type AsyncRoutes = {
    [component: string]: Es6PromiseLoader |
    Function |
    FactoryEs6PromiseLoader |
    FactoryPromise
};


type IdleCallbacks = Es6PromiseLoader |
    Function |
    FactoryEs6PromiseLoader |
    FactoryPromise;

interface WebpackModule {
    hot: {
        data?: any,
        idle: any,
        accept(dependencies?: string | string[], callback?: (updatedDependencies?: any) => void): void;
        decline(deps?: any | string | string[]): void;
        dispose(callback?: (data?: any) => void): void;
        addDisposeHandler(callback?: (data?: any) => void): void;
        removeDisposeHandler(callback?: (data?: any) => void): void;
        check(autoApply?: any, callback?: (err?: Error, outdatedModules?: any[]) => void): void;
        apply(options?: any, callback?: (err?: Error, outdatedModules?: any[]) => void): void;
        status(callback?: (status?: string) => void): void | string;
        removeStatusHandler(callback?: (status?: string) => void): void;
    };
}


interface WebpackRequire {
    (id: string): any;
    (paths: string[], callback: (...modules: any[]) => void): void;
    ensure(ids: string[], callback: (req: WebpackRequire) => void, chunkName?: string): void;
    context(directory: string, useSubDirectories?: boolean, regExp?: RegExp): WebpackContext;
}

interface WebpackContext extends WebpackRequire {
    keys(): string[];
}

interface ErrorStackTraceLimit {
    stackTraceLimit: number;
}


// Extend typings
interface NodeRequire extends WebpackRequire { }
interface ErrorConstructor extends ErrorStackTraceLimit { }
interface NodeRequireFunction extends Es6PromiseLoader { }
interface NodeModule extends WebpackModule { }
interface Global extends GlobalEnvironment { }

function baseUrl(): string {
    const base = document.querySelector('base');
    return base ? base.getAttribute('href') : '/';
}

export function uiUrl(uiPath: string): string {
    return baseUrl() + uiPath;
}

export function uiUrlWithParams(uiPath: string, params: URLSearchParams): string {
    return baseUrl() + uiPath + '?' + params.toString();
}

function getRootPath(): string {
    const metaRootPath = document.querySelector('meta[name="argo-root-path"]');
    return metaRootPath?.getAttribute('content') ?? '/';
}

export function apiUrl(apiPath: string): string {
    const rootPath = getRootPath();
    if (rootPath && rootPath !== '/') {
        const normalizedRootPath = rootPath.endsWith('/') ? rootPath : rootPath + '/';
        const normalizedApiPath = apiPath.startsWith('/') ? apiPath.slice(1) : apiPath;
        return `${normalizedRootPath}${normalizedApiPath}`;
    }
    const normalizedApiPath = apiPath.startsWith('/') ? apiPath : `/${apiPath}`;
    return normalizedApiPath;
}

export function absoluteUrl(path: string): string {
    const base = document.baseURI.endsWith('/') ? document.baseURI : document.baseURI + '/';
    return `${base}${path}`;
}

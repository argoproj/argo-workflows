function baseUrl(): string {
    const base = document.querySelector('base');
    return base ? base.getAttribute('href') : '/';
}

export function uiUrl(uiPath: string): string {
    return baseUrl() + uiPath;
}

export function apiUrl(apiPath: string): string {
    return `${baseUrl()}api${apiPath}`;
}

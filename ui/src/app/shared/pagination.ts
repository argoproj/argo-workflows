export interface Pagination {
    offset?: string;
    limit?: number;
    nextOffset?: string;
}

export function parseLimit(str: string) {
    const v = parseInt(str, 10);
    return !isNaN(v) ? v : null;
}

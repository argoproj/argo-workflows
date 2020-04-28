import {isNaN} from 'formik';

export interface Pagination {
    offset?: string;
    limit: number;
    nextOffset?: string;
}

export const defaultPaginationLimit = 10;

export function parseLimit(str: string) {
    const v = parseInt(str, 10);
    return isNaN(v) ? defaultPaginationLimit : v;
}

import {isNaN} from 'formik';

export interface Pagination {
    offset?: string;
    limit: number;
    nextOffset?: string;
}

export function parseLimit(str: string) {
    const v = parseInt(str);
    return isNaN(v) ? 10 : v;
}

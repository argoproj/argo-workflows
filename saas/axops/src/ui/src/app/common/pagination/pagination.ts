export class Pagination {
    limit: number;
    offset?: number;
    listLength?: number;
    hasMore?: boolean;
}

export class TimeRangePagination extends Pagination {
    maxTime: number;
}

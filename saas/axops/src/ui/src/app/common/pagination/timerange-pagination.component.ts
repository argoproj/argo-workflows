import { Component, Input, Output, EventEmitter, OnChanges, ViewChild } from '@angular/core';
import { DropDownComponent } from 'argo-ui-lib/src/components';

import { TimeRangePagination } from '../pagination/pagination';

@Component({
    selector: 'ax-timerange-pagination',
    templateUrl: './pagination.html',
    styles: [require('./pagination.scss')],
})
export class TimerangePaginationComponent implements OnChanges {
    @Input()
    public pagination: TimeRangePagination;

    @Input()
    public position: 'left' | 'right' = 'left';

    @Input()
    public preventActions: boolean = true;

    @Input()
    public isChangeableLimit: boolean = true;

    @Output()
    public onPaginationChange: EventEmitter<any> = new EventEmitter();

    @ViewChild('itemsPerPageDropdown')
    public itemsPerPageDropdown: DropDownComponent;

    public pageNumber: number = 1;
    private maxTimesMap: Map<number, TimeRangePagination> = new Map<number, TimeRangePagination>();
    private maxTimes: number[] = [];
    private nextMaxTime: number = null;

    ngOnChanges() {
        this.addMaxTime();
    }

    public next() {
        if (this.pagination.hasMore && this.preventActions) {
            this.addMaxTime();
            this.onPaginationChange.emit(this.pagination);
        }

        this.nextMaxTime = this.pagination.maxTime;
        this.pageNumber = this.maxTimes.indexOf(this.nextMaxTime);
    }

    public before() {
        if (!this.preventActions) {
            return;
        }

        let pagination = this.maxTimesMap.get(this.maxTimes[this.maxTimes.indexOf(this.nextMaxTime) - 2]);
        // page 1
        if (pagination === undefined) {
            return;
        }

        // page 2,3...
        if (pagination && (pagination.maxTime === 0 || pagination.maxTime)) {
            this.onPaginationChange.emit(pagination);
            this.nextMaxTime = pagination.maxTime;
        }

        this.pageNumber = this.maxTimes.indexOf(pagination.maxTime);
    }

    public getState(pagination: TimeRangePagination) {
        return {
            start: this.pageNumber * pagination.limit - pagination.limit + (pagination.listLength > 0 ? 1 : 0),
            end: this.pageNumber * pagination.limit - pagination.limit + pagination.listLength,
        };
    }

    public paginationUpdate(limit: number) {
        this.pagination.limit = limit;
        this.pagination.offset = 0;
        this.itemsPerPageDropdown.close();
        this.onPaginationChange.emit(this.pagination);
    }

    public cleanPagination() {
        this.pageNumber = 1;
        this.maxTimesMap = new Map<number, TimeRangePagination>();
        this.maxTimes = [];
        this.nextMaxTime = null;
    }

    private addMaxTime() {
        if (this.maxTimes.indexOf(this.pagination.maxTime) === -1) {
            this.maxTimes.push(this.pagination.maxTime);
            this.maxTimesMap.set(this.pagination.maxTime, this.pagination);
        }
        this.nextMaxTime = this.pagination.maxTime;
        this.pageNumber = this.maxTimes.indexOf(this.nextMaxTime);
    }
}

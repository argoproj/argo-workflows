import { Component, Input, Output, EventEmitter, ViewChild } from '@angular/core';

import { DropDownComponent } from 'argo-ui-lib/src/components';
import { Pagination } from './pagination';

@Component({
    selector: 'ax-pagination',
    templateUrl: './pagination.html',
    styles: [ require('./pagination.scss') ],
})
export class PaginationComponent {
    @Input()
    public pagination: Pagination;

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

    public getState(pagination) {
        return {
            start: pagination.offset + (pagination.listLength > 0 ? 1 : 0),
            end: pagination.offset + pagination.listLength,
        };
    }

    public paginationUpdate(limit: number) {
        this.pagination.limit = limit;
        this.pagination.offset = 0;
        this.itemsPerPageDropdown.close();
        this.onPaginationChange.emit(this.pagination);
    }

    public next() {
        if (this.pagination.hasMore && this.preventActions) {
            this.pagination.offset += this.pagination.limit;
            this.onPaginationChange.emit(this.pagination);
        }
    }

    public before() {
        if (this.pagination.offset !== 0 && this.preventActions) {
            this.pagination.offset -= this.pagination.limit;
            this.onPaginationChange.emit(this.pagination);
        }
    }
}

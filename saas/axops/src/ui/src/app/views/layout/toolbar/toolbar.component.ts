import { Component, EventEmitter, Input, Output } from '@angular/core';
import { LayoutSettings } from '../';
import { DateRange } from 'argo-ui-lib/src/components';

@Component({
    selector: 'ax-toolbar',
    templateUrl: './toolbar.html',
    styles: [ require('./toolbar.scss') ],
})
export class ToolbarComponent {

    @Input()
    public settings: LayoutSettings;

    @Output()
    public onOpenBranchNavPanel: EventEmitter<any> = new EventEmitter();

    public trackByBreadcrumbItem(i: number, item: { title: string, routerLink: any[]}) {
        return `${item.title}_${item.routerLink ? item.routerLink.join(',') : ''}`;
    }

    public openNavPanel() {
        this.onOpenBranchNavPanel.emit(null);
    }

    public get dateRangeFormatted() {
        return this.settings.layoutDateRange.data.format();
    };

    public applyDateRangeSelection(dateRange: DateRange) {
        this.settings.layoutDateRange.onApplySelection(dateRange);
    }

    public toggleFilter(option) {
        if (this.settings.toolbarFilters.model.indexOf(option.value) > -1) {
            this.settings.toolbarFilters.model.splice(this.settings.toolbarFilters.model.indexOf(option.value), 1);
        } else {
            this.settings.toolbarFilters.model.push(option.value);
        }

        this.settings.toolbarFilters.onChange(this.settings.toolbarFilters.model);
    }

    public globalAddHandler() {
        if (this.settings.globalAddAction) {
            this.settings.globalAddAction();
        }
    }

    public get hasFilters(): boolean {
        return this.settings.toolbarFilters.data.filter(item => {
            return this.settings.toolbarFilters.model.indexOf(item.value) > -1;
        }).length > 0;
    }
}

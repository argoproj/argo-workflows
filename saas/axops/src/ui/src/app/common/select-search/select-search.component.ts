import { Component, OnInit, Input, HostListener, ElementRef, Output, EventEmitter } from '@angular/core';
import { NgModel } from '@angular/forms';
import { Subject } from 'rxjs/Subject';
import 'rxjs/add/operator/debounceTime';

interface SelectSearchItem {
    name: string;
    value: string;
}

@Component({
    selector: 'ax-select-search',
    templateUrl: './select-search.html',
    styles: [ require('./select-search.scss') ],
})
export class SelectSearchComponent implements OnInit {

    @Input()
    set items(items: SelectSearchItem[]) {
        this.showLoader = false;
        this.itemsSrc = items;
    }

    @Input()
    public placeholder: string;

    @Input()
    public selected: SelectSearchItem;

    @Output()
    public onSearchQuery: EventEmitter<string> = new EventEmitter<string>();

    @Output()
    public onSelect: EventEmitter<SelectSearchItem> = new EventEmitter<SelectSearchItem>();

    public itemsSrc: SelectSearchItem[] = [];
    public searchQueryValue: NgModel;
    public searchSubject: Subject<string> = new Subject<string>();
    public showLoader: boolean;
    private isActive: boolean = false;

    constructor(private el: ElementRef) {
    }

    public ngOnInit() {
        this.searchSubject
            .debounceTime(500)
            .subscribe(value => {
                this.showLoader = true;
                this.onSearchQuery.next(value);
            });
    }

    public toggle() {
        this.isActive = !this.isActive;
    }

    public select(item: SelectSearchItem) {
        this.selected = item;
        this.isActive = false;
        this.onSelect.next(item);
    }

    public searchQuery(value: string) {
       this.searchSubject.next(value);
    }

    @HostListener('document:click', ['$event'])
    public onClick(event) {
        if (!this.el.nativeElement.contains(event.target)) {
            this.isActive = false;
        }
    }
}

import {Component, Output, EventEmitter} from '@angular/core';
import {FormGroup, FormControl} from '@angular/forms';

@Component({
    selector: 'ax-search-input',
    templateUrl: './search-input.html',
    styles: [ require('./search-input.scss') ],
})
export class SearchInputComponent {
    @Output() update = new EventEmitter();
    private searchForm: FormGroup;

    constructor() {
        this.searchForm = new FormGroup({
            searchInput: new FormControl('')
        });
    }

    search(value) {
        this.update.emit(value.searchInput);
    }
}

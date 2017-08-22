import * as _ from 'lodash';
import { Component, Output, Input, EventEmitter, ElementRef, OnInit, HostListener } from '@angular/core';
import { Location } from '@angular/common';
import { FormControl } from '@angular/forms';

@Component({
    selector: 'ax-input-select',
    templateUrl: './input-select.html',
    styles: [ require('./input-select.scss') ],
})

export class InputSelectComponent implements OnInit {

    @Input()
    limit: number = 20;

    @Input()
    placeholder: string = '';

    @Input('select-list')
    selectList: { 'key': string, 'add-date': number }[] = [];

    @Input('cache-inputs')
    cacheInputs: boolean = false;

    @Input()
    inputText: string = '';

    @Input('external-source')
    externalSource: boolean = false;
    @Input('debounce-time')
    debounceTime: number = 1000;
    @Input('input-wide')
    inputWide: string = ''; // wide, medium, small
    @Input()
    icon: string = ''; // you can use one of font awesome icons i.e. 'fa-search'. If empty icone is not displayed.
    @Input('input-class')
    inputClass: string = ''; // your customized style class for input

    @Output()
    update = new EventEmitter();
    @Output()
    refresh = new EventEmitter();
    @Output()
    onSelect = new EventEmitter();

    private selectedItemIndex: number = -1;
    private isDropdownVisible: boolean = false;
    private currentStoreVersion: number = 1;
    private currentStoreKey: string = '';
    private inputControl = new FormControl();

    constructor(private elementRef: ElementRef, private location: Location) {
    }

    ngOnInit() {
        this.currentStoreKey = `ax-input-select${this.location.path()}`;
        this.setStoredList();

        if (this.externalSource) {
            this.inputControl.valueChanges
                .debounceTime(this.debounceTime)
                .subscribe(() => {
                    this.refresh.emit(this.inputText);
                });
        }
    }

    setStoredList() {
        let currentStoreValue = localStorage.getItem(this.currentStoreKey);
        let store: Map<number, any[]> = (currentStoreValue && !_.isEmpty(JSON.parse(currentStoreValue))) ?
            new Map<number, any[]>(JSON.parse(currentStoreValue)) :
            new Map<number, any[]>();

        this.selectList = store.get(this.currentStoreVersion) || [];
        Array.from(store.keys()).forEach(key => {
            if (key < this.currentStoreVersion) {
                store.delete(key);
            }
        });
    }

    selectElement(index) {
        this.selectedItemIndex = index;
    }

    selectItem(item) {
        this.inputText = item;

        // set focus on input after mouse select
        $(this.elementRef.nativeElement).find('input')[0].focus();

        this.setDropdownVisibility(false);
        this.onSelect.emit(item);
    }

    isSelected(index) {
        return this.selectedItemIndex === index;
    }

    setDropdownVisibility(value) {
        this.isDropdownVisible = value;
        if (!value) {
            this.selectedItemIndex = -1;
        }
    }

    selectListIsNotEmpty() {
        return this.selectList.length !== 0;
    }

    updateList(value) {
        this.removeOldestOverTheLimit();

        if (value && _.findIndex(this.selectList, (item) => { return item.key === value; }) === -1) {
            this.selectList.push({ 'key': this.inputText, 'add-date': new Date().getTime() });
            this.selectList = _.sortBy(this.selectList, 'key');

            let store: Map<number, any[]> = new Map<number, any[]>();
            store.set(this.currentStoreVersion, this.selectList);

            if (this.cacheInputs) {
                // save selectList for current path in local storage
                localStorage.setItem(this.currentStoreKey, JSON.stringify(Array.from(store.entries())));
            }
        }
    }

    removeOldestOverTheLimit() {
        if (this.selectList.length >= this.limit) {
            let minDateItem = _.minBy(this.selectList, 'add-date');
            let indexToRemove = this.selectList.indexOf(minDateItem);
            this.selectList.splice(indexToRemove, 1);
        }
    }

    // keyboard click event
    keyEvent(event) {
        this.setDropdownVisibility(true);

        if (event.keyCode === 13) { // enter button

            if (this.selectedItemIndex > -1) {
                let item = this.selectList[this.selectedItemIndex];
                // User has selected an item from list
                this.updateList(item.key);
                // fire select event as user selected something from the list
                this.onSelect.emit(item.key);
                this.inputText = item.key;
            } else if (this.inputText !== '' && this.selectedItemIndex === -1) {
                // the item is added by user so we can add it
                this.updateList(this.inputText);
                this.update.emit(this.inputText);
            } else if (this.inputText === '') {
                this.update.emit(this.inputText);
            }
            // Hide the drop down
            this.setDropdownVisibility(false);
        }

        if (event.keyCode === 40) { // down arrow
            this.selectedItemIndex = this.selectedItemIndex < this.selectList.length - 1 ?
                this.selectedItemIndex + 1 : this.selectList.length - 1;
        }

        if (event.keyCode === 38) { // up arrow
            this.selectedItemIndex = this.selectedItemIndex > -1 ? this.selectedItemIndex - 1 : -1;
        }
    }

    isInputTextInItem(itemKey, inputText) {
        let lowercaseValue = itemKey.toLocaleLowerCase();
        let lowercaseSearchString = inputText.toLowerCase();
        return !lowercaseValue.includes(lowercaseSearchString);
    }

    isListEmpty(inputText) {
        let elementsOnVisibleList = _.find(this.selectList, (item) => {
            if ((item.key.toLowerCase()).includes(inputText.toLowerCase())) {
                return item;
            }
        });
        return elementsOnVisibleList === undefined;
    }

    // click outside dropdown and input
    @HostListener('document:click', ['$event'])
    onClick(event) {
        if (!this.elementRef.nativeElement.contains(event.target)) {
            this.setDropdownVisibility(false);
        }
    }
}

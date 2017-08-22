import {EventEmitter, Output} from '@angular/core';

export class LoaderInBackgroundService {
    @Output() show: EventEmitter<any> = new EventEmitter();
    @Output() hide: EventEmitter<any> = new EventEmitter();
}

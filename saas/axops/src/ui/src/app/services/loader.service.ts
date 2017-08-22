import {EventEmitter} from '@angular/core';
export class LoaderService {
    public show: EventEmitter<any> = new EventEmitter();
    public hide: EventEmitter<any> = new EventEmitter();
}

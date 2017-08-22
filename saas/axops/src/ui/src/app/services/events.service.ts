import {Output, EventEmitter} from '@angular/core';

export class EventsService {
    @Output() showNavigation: EventEmitter<any> = new EventEmitter();
    @Output() hideNavigation: EventEmitter<any> = new EventEmitter();
    public modal: EventEmitter<any> = new EventEmitter();
}

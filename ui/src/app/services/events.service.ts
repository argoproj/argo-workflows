import {Output, EventEmitter} from '@angular/core';

export class EventsService {
  @Output() setPageTitle: EventEmitter<any> = new EventEmitter();
}

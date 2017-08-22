import { Injectable } from '@angular/core';
import {BehaviorSubject} from 'rxjs/BehaviorSubject';

@Injectable()
export class SharedService {

    updateSource: BehaviorSubject<object> = new BehaviorSubject({});

    // tslint:disable-next-line:no-empty
    constructor() {}
}

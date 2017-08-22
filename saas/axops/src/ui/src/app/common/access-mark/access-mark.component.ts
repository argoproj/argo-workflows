import {Component, Input} from '@angular/core';

@Component({
    selector: 'ax-access-mark',
    templateUrl: './access-mark.html',
    styles: [ require('./access-mark.scss') ],
})
export class AccessMarkComponent {
    @Input()
    public isEnabled: boolean = false;
}

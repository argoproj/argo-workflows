import { Component, Input } from '@angular/core';
import { User } from '../../model';

@Component({
    selector: 'ax-avatar',
    templateUrl: './avatar.html',
    styles: [ require('./avatar.scss') ],
})
export class AvatarComponent {
    public initials: string = '';
    public color: string;

    private colors: string[] = [
        '#BBA2A7',
        '#98B0A4',
        '#B4A793',
        '#9FA9BD',
        '#ACA0BB',
        '#95B7C3',
        '#CAA894',
        '#B1AFB0',
        '#C3C3C3',
    ];

    @Input() public set committer(value: string) {
        if (value) {
            // Depend that committer name is one or two pieces we are displaying single letter or first name and surname letters
            let username = value.substring(0, value.indexOf('<') > 0 ? value.indexOf('<') : value.length - 1).trim();
            this.initials = username.indexOf(' ') === -1 ? username[0] : username[0] + username[username.indexOf(' ') + 1];
            this.color = this.colors[this.initials[0].charCodeAt(0) % 9];
        }
    }

    @Input() public set user(value: User) {
        this.initials = '';
        if (value && value.username) {
            this.initials = (value.first_name && value.last_name) ?
                `${value.first_name[0]}${value.last_name[0]}` : value.username[0];
            this.color = this.colors[this.initials[0].charCodeAt(0) % 9];
        }
    }

    @Input() public size: number = 40;
}

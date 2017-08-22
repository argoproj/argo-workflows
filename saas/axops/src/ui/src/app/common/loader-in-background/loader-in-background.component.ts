import {Component} from '@angular/core';

import {LoaderInBackgroundService} from '../../services';

@Component({
    selector: 'ax-loader-in-background',
    templateUrl: './loader-in-background.html',
    styles: [ require('./loader-in-background.scss') ],
})
export class LoaderInBackgroundComponent {
    private loaderInBackgroundVisible: boolean = false;
    private message: string = '';

    constructor(private loaderInBackgroundService: LoaderInBackgroundService) {
        loaderInBackgroundService.show.subscribe((response) => {
            this.message = response.message;
            this.loaderInBackgroundVisible = true;
        });

        loaderInBackgroundService.hide.subscribe((response) => {
            this.message = response.message;
            this.loaderInBackgroundVisible = false;
        });
    }
}

import { Component, ViewEncapsulation } from '@angular/core';
import { TranslateService } from 'ng2-translate/ng2-translate';

@Component({
    selector: 'ax-app',
    templateUrl: './app.html',
    encapsulation: ViewEncapsulation.None,
    styles: [
        require('../assets/styles/main.scss').toString(),
        require('./app.scss').toString(),
    ],
})
export class AppComponent {
    constructor(public _translate: TranslateService) {
        // use navigator lang if available
        let userLang = navigator.language.split('-')[0];
        userLang = /(fr|en|pl)/gi.test(userLang) ? userLang : 'en';
        userLang = 'en';

        // this trigger the use of the french or english language after setting the translations
        this._translate.use(userLang);
    }
}

import { Component } from '@angular/core';

import { HasLayoutSettings } from '../../layout';

@Component({
    selector: 'ax-settings-overview',
    templateUrl: './settings-overview.html',
    styles: [ require('./settings-overview.scss') ]
})
export class SettingsOverviewComponent implements HasLayoutSettings {

    public get layoutSettings() {
        return {
            pageTitle: 'Settings',
            hiddenToolbar: true,
        };
    }
}

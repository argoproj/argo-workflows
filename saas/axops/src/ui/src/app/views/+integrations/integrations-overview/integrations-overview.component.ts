import { Component } from '@angular/core';

import { HasLayoutSettings } from '../../layout';

@Component({
    selector: 'ax-integrations-overview',
    templateUrl: './integrations-overview.html',
    styles: [ require('./integrations-overview.scss') ]
})
export class IntegrationsOverviewComponent implements HasLayoutSettings {

    public get layoutSettings() {
        return {
            pageTitle: 'Integrations',
            hiddenToolbar: true,
        };
    }
}

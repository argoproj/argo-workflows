import {Component, ViewChild} from '@angular/core';
import {RouterOutlet} from '@angular/router';

import {LayoutSettings, HasLayoutSettings} from '../layout/layout.component';

@Component({
    selector: 'ax-service-catalog-root',
    templateUrl: './service-catalog-root.html',
})
export class ServiceCatalogRootComponent implements HasLayoutSettings {
    @ViewChild(RouterOutlet)
    routerOutlet: RouterOutlet;

    get layoutSettings(): LayoutSettings {
        return this.routerOutlet.component;
    }
}

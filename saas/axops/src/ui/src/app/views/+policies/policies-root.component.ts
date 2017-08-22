import {Component, ViewChild} from '@angular/core';
import {RouterOutlet} from '@angular/router';

import {HasLayoutSettings, LayoutSettings} from '../layout/layout.component';

@Component({
    selector: 'ax-policies-root',
    templateUrl: './policies.html',
})

export class PoliciesRootComponent implements HasLayoutSettings {
    @ViewChild(RouterOutlet)
    routerOutlet: RouterOutlet;

    get layoutSettings(): LayoutSettings {
        return this.routerOutlet.component;
    }
}

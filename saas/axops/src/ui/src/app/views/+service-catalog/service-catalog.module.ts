import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

import { decorateRouteDefs } from '../../app.routes';

import { ServiceCatalogOverviewComponent } from './service-catalog-overview/service-catalog-overview.component';
import { ServiceCatalogRootComponent } from './service-catalog-root.component';
import { ServiceHistoryComponent } from './service-history/service-history.component';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common';

export const routes = [{
    path: '',
    component: ServiceCatalogRootComponent,
    children: [
        { path: 'overview', component: ServiceCatalogOverviewComponent, terminal: true },
        { path: 'history/:id', component: ServiceHistoryComponent, terminal: true },
    ]
}];

@NgModule({
    declarations: [
        ServiceCatalogOverviewComponent,
        ServiceCatalogRootComponent,
        ServiceHistoryComponent
    ],
    imports: [
        PipesModule,
        ComponentsModule,
        CommonModule,
        RouterModule.forChild(decorateRouteDefs(routes)),
    ]
})
export default class ServiceCatalogModule {
}

import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';

import { GlobalSearchResultsComponent } from './global-search-results.component';
import { decorateRouteDefs } from '../../app.routes';

import { ComponentsModule } from '../../common/components.module';
import { PipesModule } from '../../pipes/pipes.module';
import { CommitsListComponent } from './commits-list/commits-list.component';
import { TemplatesListComponent } from './templates-list/templates-list.component';
import { JobsListComponent } from './jobs-list/jobs-list.component';
import { ApplicationsListComponent } from './applications-list/applications-list.component';
import { DeploymentsListComponent } from './deployments-list/deployments-list.component';
import { GlobalSearchFilterComponent } from './global-search-filter/global-search-filter.component';

export const routes = [
    { path: '', component: GlobalSearchResultsComponent, terminal: true },
    { path: ':keyword', component: GlobalSearchResultsComponent, terminal: true },
];

@NgModule({
    declarations: [
        GlobalSearchResultsComponent,
        CommitsListComponent,
        TemplatesListComponent,
        JobsListComponent,
        ApplicationsListComponent,
        GlobalSearchFilterComponent,
        DeploymentsListComponent,
    ],
    imports: [
        ComponentsModule,
        CommonModule,
        PipesModule,
        FormsModule,
        RouterModule.forChild(decorateRouteDefs(routes)),
    ]
})
export default class GlobalSearchModule {
}

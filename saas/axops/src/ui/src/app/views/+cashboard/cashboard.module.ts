import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

import { decorateRouteDefs } from '../../../app/app.routes';

import { CashboardComponent } from './cashboard/cashboard.component';
import { CashboardItemDetailsComponent } from './cashboard-item-details/cashboard-item-details.component';
import { CashboardSummariesComponent } from './cashboard-summaries/cashboard-summaries.component';
import { CashboardTypeDetailsComponent } from './cashboard-type-details/cashboard-type-details.component';
import { SpendingsChartComponent } from './spendings-chart/spendings-chart.component';

import { ComponentsModule } from '../../common';
import { PipesModule } from '../../pipes';


export const routes = [
    { path: '', component: CashboardComponent },
    { path: 'details/:type', component: CashboardTypeDetailsComponent },
    { path: 'details/:type/:name', component: CashboardItemDetailsComponent },
];

@NgModule({
    declarations: [
        SpendingsChartComponent,
        CashboardComponent,
        CashboardSummariesComponent,
        CashboardTypeDetailsComponent,
        CashboardItemDetailsComponent,
    ],
    imports: [
        ComponentsModule,
        PipesModule,
        CommonModule,
        RouterModule.forChild(decorateRouteDefs(routes)),
    ]
})
export default class CashboardModule {
}

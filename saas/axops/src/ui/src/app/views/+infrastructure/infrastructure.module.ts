import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

import { decorateRouteDefs } from '../../app.routes';

import { InfrastructureComponent } from './infrastructure.component';
import { SpendingLineChartComponent } from './spending-line-chart/spending-line-chart.component';
import { SpendingSquareChartComponent } from './spending-square-chart/spending-square-chart.component';
import { TooltipDirective } from './spending-square-chart/tooltip/tooltip.directive';

import { PipesModule } from '../../pipes';

export const routes = [{
    path: '',
    component: InfrastructureComponent,
}];


@NgModule({
    declarations: [ InfrastructureComponent, SpendingLineChartComponent, SpendingSquareChartComponent, TooltipDirective ],
    imports: [ PipesModule, CommonModule, RouterModule.forChild(decorateRouteDefs(routes, true)), ],
})
export default class InfrastructureModule {
}

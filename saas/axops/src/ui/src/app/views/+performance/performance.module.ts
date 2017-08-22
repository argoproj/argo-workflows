import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

import { decorateRouteDefs } from '../../app.routes';

import { BuildsBoxPlotChartComponent } from './builds-boxplot-chart/builds-boxplot-chart.component';
import { BuildsLineChartComponent } from './builds-line-chart/builds-line-chart.component';
import { PerformanceComponent } from './performance.component';

import { PipesModule } from '../../pipes';

export const routes = [{
    path: '', component: PerformanceComponent,
}];

@NgModule({
    declarations: [ BuildsLineChartComponent, BuildsBoxPlotChartComponent, PerformanceComponent ],
    imports: [ PipesModule, CommonModule, RouterModule.forChild(decorateRouteDefs(routes)), ],
})
export default class PerformanceModule {

}

import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';

import { decorateRouteDefs } from '../../app.routes';
import { AppOverviewComponent } from './app-overview/app-overview.component';
import { AppDetailsComponent } from './application-details/application-details.component';
import { ApplicationPanelComponent } from './application-panel/application-panel.component';
import { StartComponent } from './actions/start.component';
import { TerminateComponent } from './actions/terminate.component';
import { StopComponent } from './actions/stop.component';
import { AppGraphComponent, AppGraphStatusIconComponent } from './app-graph/app-graph.component';
import { DeploymentDetailsComponent } from './deployment-details/deployment-details.component';
import { DeploymentHistoryCellComponent } from './deployment-history-cell/deployment-history-cell.component';
import { AppSpendingsChartComponent } from './app-spendings-chart/app-spendings-chart.component';
import { DeploymentHistoryComponent } from './deployment-history/deployment-history.component';
import { DeploymentHistoryDetailsComponent } from './deployment-history-details/deployment-history-details.component';
import { ApplicationStatusBarComponent } from './application-status-bar/application-status-bar.component';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common/components.module';

export const routes = [
    { path: '', component: AppOverviewComponent, terminal: true },
    { path: 'details/:id', component: AppDetailsComponent }
];

@NgModule({
    declarations: [
        AppOverviewComponent,
        ApplicationPanelComponent,
        AppDetailsComponent,
        StartComponent,
        TerminateComponent,
        StopComponent,
        AppSpendingsChartComponent,
        AppGraphComponent,
        DeploymentDetailsComponent,
        AppGraphStatusIconComponent,
        DeploymentHistoryCellComponent,
        DeploymentHistoryComponent,
        ApplicationStatusBarComponent,
        DeploymentHistoryDetailsComponent,
    ],
    imports: [
        PipesModule,
        FormsModule,
        ReactiveFormsModule,
        RouterModule.forChild(decorateRouteDefs(routes, true)),
        CommonModule,
        RouterModule,
        ComponentsModule,
    ],
})
export default class ApplicationsModule {

}


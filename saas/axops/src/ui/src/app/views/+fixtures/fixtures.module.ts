import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { decorateRouteDefs } from '../../../app/app.routes';

import { ComponentsModule } from '../../common';
import { PipesModule } from '../../pipes';
import { TimelineComponentsModule } from '../+timeline/timeline-components.module';
import { FixtureClassesComponent } from './fixture-classes/fixture-classes.component';
import { FixtureInstancesComponent } from './fixture-instances/fixture-instances.component';
import { FixtureTemplatePanelComponent } from './fixture-template-panel/fixture-template-panel.component';
import { FixtureInstanceFormComponent } from './fixture-instance-form/fixture-instance-form.component';
import { FixtureInstanceStatusComponent } from './fixture-instance-status/fixture-instance-status.component';
import { FixtureInstanceDetailsComponent } from './fixture-instance-details/fixture-instance-details.component';
import { FixtureActionLaunchPanelComponent } from './fixture-action-launch-panel/fixture-action-launch-panel.component';
import { FixturesViewService } from './fixtures.view-service';
import { FixtureUsageChartComponent } from './fixture-usage-chart/fixture-usage-chart.component';
import { FixtureInstanceAttributesPanelComponent  } from './fixture-instance-attributes-panel/fixture-instance-attributes-panel.component';

export const routes = [
    { path: '', component: FixtureClassesComponent, terminal: true },
    { path: ':id', component: FixtureInstancesComponent, terminal: true },
    { path: ':id/details/:instanceId', component: FixtureInstanceDetailsComponent, terminal: true },
];

@NgModule({
    declarations: [
        FixtureTemplatePanelComponent,
        FixtureClassesComponent,
        FixtureInstancesComponent,
        FixtureInstanceFormComponent,
        FixtureInstanceStatusComponent,
        FixtureInstanceDetailsComponent,
        FixtureUsageChartComponent,
        FixtureActionLaunchPanelComponent,
        FixtureInstanceAttributesPanelComponent,
    ],
    providers: [ FixturesViewService ],
    imports: [
        ComponentsModule,
        PipesModule,
        FormsModule,
        ReactiveFormsModule,
        RouterModule.forChild(decorateRouteDefs(routes, true)),
        CommonModule,
        TimelineComponentsModule,
    ],
})
export default class FixturesModule {
}

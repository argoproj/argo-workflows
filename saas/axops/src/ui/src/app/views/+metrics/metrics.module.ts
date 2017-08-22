import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { TestDashboardComponent } from './test-dashboard/test-dashboard.component';
import { CustomViewComponent } from './custom-view/custom-view.component';

import { decorateRouteDefs } from '../../app.routes';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common';

export const routes = [
    { path: '', component: TestDashboardComponent },
];

@NgModule({
    declarations: [
        TestDashboardComponent,
        CustomViewComponent,
    ],
    imports: [
        FormsModule,
        ReactiveFormsModule,
        ComponentsModule,
        CommonModule,
        PipesModule,
        RouterModule.forChild(decorateRouteDefs(routes)),
    ]
})
export default class MetricsModule {

}

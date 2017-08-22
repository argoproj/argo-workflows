import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { decorateRouteDefs } from '../../../app/app.routes';
import { ComponentsModule } from '../../common';
import { PipesModule } from '../../pipes';

import { ConfigManagementOverviewComponent } from './config-management-overview/config-management-overview.component';
import { ConfigManagementPanelComponent } from './config-management-panel/config-management-panel.component';

export const routes = [
    { path: '', component: ConfigManagementOverviewComponent },
];

@NgModule({
    declarations: [
        ConfigManagementOverviewComponent,
        ConfigManagementPanelComponent,
    ],
    imports: [
        ComponentsModule,
        PipesModule,
        FormsModule,
        ReactiveFormsModule,
        RouterModule.forChild(decorateRouteDefs(routes)),
        CommonModule,
    ],
})
export default class ConfigManagementModule {
}

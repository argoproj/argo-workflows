import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { CustomFormsModule } from 'ng2-validation/dist/index';

import { decorateRouteDefs } from '../../../app/app.routes';

import { ComponentsModule } from '../../common/components.module';
import { PipesModule } from '../../pipes/pipes.module';
import { VolumesOverviewComponent } from './volumes-overview/volumes-overview.component';
import { VolumeDetailsComponent } from './volume-details/volume-details.component';
import { VolumeAddPanelComponent } from './volume-add-panel/volume-add-panel.component';
import { VolumeEditPanelComponent } from './volume-edit-panel/volume-edit-panel.component';
import { VolumeFormWidgetComponent } from './volume-form-widget/volume-form-widget.component';

export const routes = [
    { path: '', component: VolumesOverviewComponent, terminal: true },
    { path: ':id', component: VolumeDetailsComponent, terminal: true },
];

@NgModule({
    declarations: [
        VolumesOverviewComponent,
        VolumeDetailsComponent,
        VolumeAddPanelComponent,
        VolumeFormWidgetComponent,
        VolumeEditPanelComponent,
    ],
    imports: [
        ComponentsModule,
        FormsModule,
        CustomFormsModule,
        RouterModule.forChild(decorateRouteDefs(routes)),
        CommonModule,
        PipesModule,
    ]
})
export default class VolumesModule {
}

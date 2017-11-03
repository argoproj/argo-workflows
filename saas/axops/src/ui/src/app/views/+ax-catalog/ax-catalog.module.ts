import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

import { ComponentsModule } from '../../common';
import { decorateRouteDefs } from '../../../app/app.routes';

import { AxCatalogOverviewComponent } from './ax-catalog-overview/ax-catalog-overview.component';
import { ProjectIconComponent } from './project-icon/project-icon.component';
import { ProjectLaunchButtonComponent } from './project-launch-button/project-launch-button.component';
import { ProjectDetailsComponent } from './project-details/project-details.component';
import { ProjectDetailsPanelComponent } from './project-details-panel/project-details-panel.component';
import { PipesModule } from '../../pipes';

export const routes = [
    {
        path: '', component: AxCatalogOverviewComponent, terminal: true,
    },
    {
        path: ':id', component: ProjectDetailsComponent, terminal: true,
    }
];

@NgModule({
    declarations: [
        AxCatalogOverviewComponent,
        ProjectIconComponent,
        ProjectLaunchButtonComponent,
        ProjectDetailsComponent,
        ProjectDetailsPanelComponent,
    ],
    imports: [
        CommonModule,
        ComponentsModule,
        RouterModule,
        PipesModule,
    ],
    exports: [
        AxCatalogOverviewComponent,
        ProjectDetailsComponent,
        ProjectDetailsPanelComponent,
    ]
})
export class AxCatalogComponentsModule {

}

@NgModule({
    imports: [
        AxCatalogComponentsModule,
        RouterModule.forChild(decorateRouteDefs(routes, true)),
    ],
})
export default class AxCatalogModule {

}

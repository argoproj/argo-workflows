import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common';
import { decorateRouteDefs } from '../../../app/app.routes';

import { IntroductionComponent } from './introduction/introduction.component';
import { AxCatalogWrapperComponent } from './ax-catalog-wrapper/ax-catalog-wrapper.component';
import { IntegrationsWrapperComponent } from './integrations-wrapper/integrations-wrapper.component';

import { AxCatalogComponentsModule } from '../+ax-catalog/ax-catalog.module';
import { AxCatalogOverviewComponent } from '../+ax-catalog/ax-catalog-overview/ax-catalog-overview.component';
import { ProjectDetailsComponent } from '../+ax-catalog/project-details/project-details.component';

import { IntegrationsComponentsModule } from '../+integrations/integrations.module';
import { IntegrationsComponent } from '../+integrations/integrations/integrations.component';

import { PlaygroundComponent } from './playground/playground.component';

export const routes = [
    { path: '', component: IntroductionComponent, terminal: true },
    {
        path: 'ax-catalog',
        component: AxCatalogWrapperComponent,
        children: [{
            path: '', component: AxCatalogOverviewComponent, terminal: true,
        }, {
            path: ':id', component: ProjectDetailsComponent, terminal: true,
        }]
    }, {
        path: 'integrations',
        component: IntegrationsWrapperComponent,
        children: [{
            path: '', component: IntegrationsComponent, terminal: true,
        }]
    }, {
        path: 'playground', component: PlaygroundComponent, terminal: true,
    },
];

@NgModule({
    declarations: [
        IntroductionComponent,
        AxCatalogWrapperComponent,
        IntegrationsWrapperComponent,
        PlaygroundComponent,
    ],
    imports: [
        FormsModule,
        ReactiveFormsModule,
        PipesModule,
        CommonModule,
        ComponentsModule,
        AxCatalogComponentsModule,
        IntegrationsComponentsModule,
        RouterModule.forChild(decorateRouteDefs(routes)),
    ]
})
export default class FirstUserExperienceModule {

}

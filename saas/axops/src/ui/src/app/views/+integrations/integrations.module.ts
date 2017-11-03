import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';

import { GitPanelComponent } from './git-panel/git-panel.component';
import { ToolPanelComponent } from './tool-panel/tool-panel.component';
import { SmtpPanelComponent } from './smtp-panel/smtp-panel.component';
import { NexusPanelComponent } from './nexus-panel/nexus-panel.component';
import { RegistryPanelComponent } from './registry-panel/registry-panel.component';
import { JiraPanelComponent } from './jira-panel/jira-panel.component';
import { IntegrationsComponent } from './integrations/integrations.component';
import { SlackPanelComponent } from './slack-panel/slack-panel.component';
import { decorateRouteDefs } from '../../app.routes';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common';
import { IntegrationsOverviewComponent } from './integrations-overview/integrations-overview.component';

export const routes = [
    { path: '', component: IntegrationsComponent, terminal: true },
    { path: 'overview', component: IntegrationsOverviewComponent, terminal: true },
];

@NgModule({
    declarations: [
        IntegrationsComponent,
        IntegrationsOverviewComponent,
        GitPanelComponent,
        ToolPanelComponent,
        SmtpPanelComponent,
        NexusPanelComponent,
        RegistryPanelComponent,
        SlackPanelComponent,
        JiraPanelComponent,
    ],
    exports: [
        IntegrationsComponent,
    ],
    imports: [
        FormsModule,
        ReactiveFormsModule,
        PipesModule,
        CommonModule,
        ComponentsModule,
        RouterModule,
    ]
})
export class IntegrationsComponentsModule {

}

@NgModule({
    imports: [
        IntegrationsComponentsModule,
        RouterModule.forChild(decorateRouteDefs(routes, true)),
    ]
})
export default class IntegrationsModule {

}

import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { decorateRouteDefs } from '../../app.routes';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common';

import { SettingsOverviewComponent } from './settings-overview/settings-overview.component';
import { SystemSettingsComponent } from './system-settings/system-settings.component';
import { SpotInstancesPanelComponent } from './system-settings/spot-instances-panel/spot-instances-panel.component';
import { DomainManagementComponent } from './domain-mgmt/domain.component';
import { NotificationManagementComponent } from './notification-management/notification-management.component';
import { NotificationCreationPanelComponent } from './notification-management/notification-creation-panel/notification-creation-panel.component';
import { ArtifactRetentionPolicyComponent } from './artifact-retention-policy/artifact-retention-policy.component';
import { RetentionPolicyRowComponent } from './artifact-retention-policy/retention-policy-row/retention-policy-row.component';


export const routes = [
    { path: 'overview', component: SettingsOverviewComponent },
    { path: 'system', component: SystemSettingsComponent },
    { path: 'domain-management', component: DomainManagementComponent },
    { path: 'notification-management', component: NotificationManagementComponent },
    { path: 'artifact-retention-policy', component: ArtifactRetentionPolicyComponent },
];

@NgModule({
    declarations: [
        SettingsOverviewComponent,
        SystemSettingsComponent,
        SpotInstancesPanelComponent,
        DomainManagementComponent,
        NotificationManagementComponent,
        NotificationCreationPanelComponent,
        ArtifactRetentionPolicyComponent,
        RetentionPolicyRowComponent,
    ],
    imports: [
        PipesModule,
        CommonModule,
        ComponentsModule,
        FormsModule,
        ReactiveFormsModule,
        RouterModule.forChild(decorateRouteDefs(routes)),
    ]
})
export default class SettingsComponents {

}

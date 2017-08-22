import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

import { ComponentsModule } from '../../../common/components.module';
import { PipesModule } from '../../../pipes';
import { NotificationsPanelComponent } from './notifications-panel/notifications-panel.component';
import { NotificationSummaryComponent } from './notification-summary/notification-summary.component';
import { NotificationDetailsComponent } from './notification-details/notification-details.component';

@NgModule({
    declarations: [
        NotificationsPanelComponent,
        NotificationSummaryComponent,
        NotificationDetailsComponent,
    ],
    imports: [
        CommonModule,
        ComponentsModule,
        RouterModule,
        PipesModule,
    ],
    exports: [
        NotificationsPanelComponent,
    ]
})
export class NotificationsCenterModule {
}

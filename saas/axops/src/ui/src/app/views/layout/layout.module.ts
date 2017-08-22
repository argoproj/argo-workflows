import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { LayoutComponent } from './layout.component';
import {
    NavigationPanelComponent,
    PanelBodyDirective,
    PanelFooterDirective,
    PanelHeaderDirective
} from './navigation/navigation-panel/navigation-panel.component';
import { NavigationComponent } from './navigation/navigation.component';
import { TopBarComponent } from './top-bar/top-bar.component';
import { TutorialComponent } from './tutorial/tutorial.component';
import { PlaygroundInfoComponent } from './playground-info/playground-info.component';
import { NotificationsCenterModule } from './notifications-center/notifications-center.module';
import { ToolbarComponent } from './toolbar/toolbar.component';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common';

@NgModule({
    declarations: [
        LayoutComponent,
        NavigationComponent,
        NavigationPanelComponent,
        TopBarComponent,
        PanelBodyDirective,
        PanelFooterDirective,
        PanelHeaderDirective,
        TutorialComponent,
        PlaygroundInfoComponent,
        ToolbarComponent,
    ],
    imports: [
        PipesModule,
        ComponentsModule,
        CommonModule,
        RouterModule,
        FormsModule,
        ReactiveFormsModule,
        NotificationsCenterModule,
    ],
})
export class LayoutModule {

}

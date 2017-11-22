import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { LayoutComponent } from './layout.component';
import { NavigationComponent } from './navigation/navigation.component';
import { TopBarComponent } from './top-bar/top-bar.component';
import { ToolbarComponent } from './toolbar/toolbar.component';
import { AppRoutingModule } from '../app-routing.module';

@NgModule({
  declarations: [
    LayoutComponent,
    TopBarComponent,
    NavigationComponent,
    ToolbarComponent,
  ],
  imports: [
    CommonModule,
    RouterModule,
    FormsModule,
    ReactiveFormsModule,
    AppRoutingModule,
  ],
  exports: [
    LayoutComponent
  ]
})
export class LayoutModule {

}

import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { LayoutComponent } from './layout.component';
import { NavigationComponent } from './navigation/navigation.component';
import { TopBarComponent } from './top-bar/top-bar.component';
import { AppRoutingModule } from '../app-routing.module';
import { ServicesModule } from '../services';

@NgModule({
  declarations: [
    LayoutComponent,
    TopBarComponent,
    NavigationComponent,
  ],
  imports: [
    CommonModule,
    RouterModule,
    FormsModule,
    ReactiveFormsModule,
    AppRoutingModule,
    ServicesModule
  ],
  exports: [
    LayoutComponent
  ]
})
export class LayoutModule {

}

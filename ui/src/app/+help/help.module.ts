import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';

import { BaseModule } from '../common';

import { HelpRoutingModule } from './help-routing.module';
import { HelpComponent } from './help.component';
@NgModule({
  imports: [
    CommonModule,
    HelpRoutingModule,
    BaseModule,
  ],
  declarations: [
    HelpComponent
  ]
})
export class HelpModule { }

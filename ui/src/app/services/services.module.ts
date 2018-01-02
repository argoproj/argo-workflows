import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { HttpClientModule, HTTP_INTERCEPTORS } from '@angular/common/http';
import { WorkflowsService } from './workflows.service';
import { LoaderService } from './loader.service';
import { SystemService } from './system.service';
import { Interceptor } from './interceptor';

@NgModule({
  imports: [
    CommonModule,
    HttpClientModule,
  ],
  declarations: [],
  providers: [
    SystemService,
    LoaderService,
    WorkflowsService,
    {provide: HTTP_INTERCEPTORS, useClass: Interceptor, multi: true}
  ]
})
export class ServicesModule { }

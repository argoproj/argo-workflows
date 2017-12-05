import { BrowserModule } from '@angular/platform-browser';
import { NgModule } from '@angular/core';

import 'rxjs/add/observable/interval';
import 'rxjs/add/operator/take';
import 'rxjs/add/operator/map';
import 'rxjs/add/operator/bufferCount';

import { AppRoutingModule } from './app-routing.module';

import { AppComponent } from './app.component';
import { LayoutModule } from './layout/layout.module';
import { EventsService } from './services/events.service';

@NgModule({
  declarations: [
    AppComponent
  ],
  imports: [
    BrowserModule,
    AppRoutingModule,
    LayoutModule
  ],
  providers: [
    EventsService,
  ],
  bootstrap: [AppComponent]
})
export class AppModule { }

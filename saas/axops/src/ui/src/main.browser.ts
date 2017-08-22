import { APP_BASE_HREF, LocationStrategy, PathLocationStrategy, HashLocationStrategy, CommonModule } from '@angular/common';
import { decorateModuleRef } from './app/environment';

import { TranslateModule, TranslateLoader, TranslateStaticLoader } from 'ng2-translate/ng2-translate';
import { NgModule } from '@angular/core';
import { Http } from '@angular/http';
import { platformBrowserDynamic } from '@angular/platform-browser-dynamic';
import { BrowserModule } from '@angular/platform-browser';
import { RouterModule } from '@angular/router';
import { ToasterModule } from 'angular2-toaster/angular2-toaster';

import { AppComponent } from './app/app.component';
import { ToasterService } from 'angular2-toaster/angular2-toaster';
import { ROUTERS } from './app/app.routes';

import { ComponentsModule } from './app/common';
import { ViewsModule } from './app/views/views.module';
import { ServicesModule } from './app/services';

@NgModule({
    bootstrap: [AppComponent],
    providers: [
        ToasterService,
        {
            provide: LocationStrategy,
            useClass: ENV === 'production' ? PathLocationStrategy : HashLocationStrategy,
        },
        {
            provide: APP_BASE_HREF,
            useValue: '/'
        }],
    declarations: [ AppComponent ],
    imports: [
        ToasterModule,
        CommonModule,
        BrowserModule,
        ComponentsModule,
        ViewsModule,
        ServicesModule,
        RouterModule.forRoot(ROUTERS),
        TranslateModule.forRoot({
            provide: TranslateLoader,
            useFactory: (http: Http) => new TranslateStaticLoader(http, '/assets/i18n', '.json'),
            deps: [Http]
        })
    ]
})
class AppModule {

}

platformBrowserDynamic()
    .bootstrapModule(AppModule)
    .then(decorateModuleRef)
    .catch(err => console.error(err));

if (module.hot) {
    // Reload page if any angular module has changed. CSS changes will be injected without page reload.
    module.hot.accept();
}

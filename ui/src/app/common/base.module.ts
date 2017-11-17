import { NgModule } from '@angular/core';
import { ServicesModule } from '../services';


@NgModule({
    exports: [
        ServicesModule,
    ]
})
export class BaseModule {

}

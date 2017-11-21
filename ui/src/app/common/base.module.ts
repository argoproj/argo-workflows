import { NgModule } from '@angular/core';
import { ServicesModule } from '../services';

import { GuiComponentsModule } from 'ui-lib/src/components';
import { ComponentsModule } from './components.module';

@NgModule({
    exports: [
        ServicesModule,
        GuiComponentsModule,
        ComponentsModule,
    ]
})
export class BaseModule {

}

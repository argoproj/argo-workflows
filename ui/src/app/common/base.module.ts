import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ServicesModule } from '../services';

import { GuiComponentsModule } from 'ui-lib/src/components';
import { ComponentsModule } from './components.module';

@NgModule({
    exports: [
        CommonModule,
        ServicesModule,
        GuiComponentsModule,
        ComponentsModule,
    ]
})
export class BaseModule {

}

import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

import { PipesModule } from '../../pipes';

import { decorateRouteDefs } from '../../app.routes';

import { HostsComponent } from './hosts.component';

export const routes = [{
    path: '', component: HostsComponent,
}];

@NgModule({
    declarations: [ HostsComponent ],
    imports: [ PipesModule, CommonModule, RouterModule.forChild(decorateRouteDefs(routes)), ],
})
export default class HostsModule {
}

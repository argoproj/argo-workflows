import { NgModule } from '@angular/core';
import { RouterModule } from '@angular/router';

import { decorateRouteDefs } from '../../app.routes';

import { HelpComponent } from './help.component';

export const routes = [{
    path: '', component: HelpComponent,
}];

@NgModule({
    declarations: [ HelpComponent ],
    imports: [
        RouterModule.forChild(decorateRouteDefs(routes)),
    ]
})
export default class HelpModule {

}

import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';

import { decorateRouteDefs } from '../../app.routes';

import { PoliciesOverviewComponent } from './policies-overview/policies-overview.component';
import { PoliciesRootComponent } from './policies-root.component';
import { PolicyDetailsComponent } from './policy-details/policy-details.component';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common';

export const routes = [
    {
        path: '',
        component: PoliciesRootComponent,
        children: [
            { path: 'overview', component: PoliciesOverviewComponent },
            { path: 'details/:policyId', component: PolicyDetailsComponent }
        ]
    },
];

@NgModule({
    declarations: [
        PoliciesOverviewComponent,
        PoliciesRootComponent,
        PolicyDetailsComponent,
    ],
    imports: [
        PipesModule,
        CommonModule,
        ComponentsModule,
        FormsModule,
        ReactiveFormsModule,
        RouterModule.forChild(decorateRouteDefs(routes, true)),
    ],
})
export default class PoliciesModule {
}

import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { decorateRouteDefs } from '../../app.routes';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common';

import { ManageUsersComponent } from './manage-users.component';
import { InviteUserComponent } from './invite-user/invite.component';
import { EditUserComponent } from './edit-user/edit-user.component';
import { ChangePasswordComponent } from './change-password/change-password.component';
import { SamlConfigComponent } from './saml-config/saml-config.component';
import { UserProfileComponent } from './user-profile/user-profile.component';
import { UserUtils } from './user-utils';

export const routes = [
    { path: 'overview', component: ManageUsersComponent },
    { path: 'profile/:username', component: UserProfileComponent },
];

@NgModule({
    declarations: [
        ManageUsersComponent,
        InviteUserComponent,
        EditUserComponent,
        ChangePasswordComponent,
        SamlConfigComponent,
        UserProfileComponent,
    ],
    providers: [ UserUtils ],
    imports: [
        PipesModule,
        CommonModule,
        ComponentsModule,
        RouterModule,
        FormsModule,
        ReactiveFormsModule,
        RouterModule.forChild(decorateRouteDefs(routes)),
    ]
})
export default class UserManagementModule {

}

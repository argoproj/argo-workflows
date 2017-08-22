import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { PipesModule } from '../../pipes';

import { ResetConfirmationComponent } from './reset-confirmation/reset-confirmation.component';
import { ResetPasswordComponent } from './reset-password.component';
import { ComponentsModule } from '../../common';

@NgModule({
    declarations: [ ResetPasswordComponent, ResetConfirmationComponent ],
    imports: [ FormsModule, ReactiveFormsModule, PipesModule, CommonModule, RouterModule, ComponentsModule ]
})
export class ResetPasswordModule {

}

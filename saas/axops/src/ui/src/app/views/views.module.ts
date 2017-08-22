import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { BrowserModule } from '@angular/platform-browser';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';

import { PipesModule } from '../pipes';
import { ComponentsModule } from '../common';
import { ForgotPasswordModule } from './forgot-password';
import { LoginModule } from './login';
import { ResetPasswordModule } from './reset-password';
import { SetupModule } from './setup';
import { LayoutModule } from './layout';

@NgModule({
    imports: [
        CommonModule,
        BrowserModule,
        FormsModule,
        ReactiveFormsModule,
        PipesModule,
        ComponentsModule,
        RouterModule,
        ForgotPasswordModule,
        LoginModule,
        ResetPasswordModule,
        SetupModule,
        LayoutModule,
    ]
})
export class ViewsModule {

}

import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { BrowserModule } from '@angular/platform-browser';
import { RouterModule } from '@angular/router';

import { PipesModule } from '../../pipes';

import { ForgotPasswordComponent } from './forgot-password.component';
import { ForgotConfirmationComponent } from './forgot-confirmation/forgot-confirmation.component';
import { ComponentsModule } from '../../common';


@NgModule({
    declarations: [ForgotConfirmationComponent, ForgotPasswordComponent],
    imports: [
        PipesModule,
        FormsModule,
        ReactiveFormsModule,
        CommonModule,
        BrowserModule,
        RouterModule,
        ComponentsModule,
    ],
})
export class ForgotPasswordModule {
}

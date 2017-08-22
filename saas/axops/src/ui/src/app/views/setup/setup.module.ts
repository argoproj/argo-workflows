import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';

import { EmailConfirmationComponent } from './email-confirmation/email-confirmation.component';
import { SetupComponent } from './setup.component';
import { SignUpComponent } from './sign-up/sign-up.component';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common';

@NgModule({
    declarations: [ EmailConfirmationComponent, SetupComponent, SignUpComponent],
    imports: [ CommonModule, PipesModule, RouterModule, FormsModule, ReactiveFormsModule, ComponentsModule ],
})
export class SetupModule {

}

import { NgModule } from '@angular/core';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common';

import { LoginComponent } from './login.component';

@NgModule({
    declarations: [ LoginComponent ],
    exports: [ LoginComponent ],
    imports: [ CommonModule, FormsModule, ReactiveFormsModule, PipesModule, ComponentsModule, RouterModule ]
})
export class LoginModule {
}

import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule } from '@angular/forms';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common';

import { JobStepsComponent } from './jobs-overview/job-steps/job-steps.component';
import { JobComponent } from './jobs-overview/job/job.component';
import { JobsService } from './jobs.service';

@NgModule({
    declarations: [
        JobComponent,
        JobStepsComponent,
    ],
    exports: [
        JobComponent,
        JobStepsComponent,
    ],
    providers: [
        JobsService,
    ],
    imports: [
        CommonModule,
        PipesModule,
        ComponentsModule,
        FormsModule,
        RouterModule,
    ]
})
export class TimelineComponentsModule {

}

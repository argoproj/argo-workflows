import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { RouterModule } from '@angular/router';
import { FormsModule } from '@angular/forms';

import { decorateRouteDefs } from '../../app.routes';

import { CommitBoxComponent } from './commit-box/commit-box.component';
import { JobActionsComponent } from './job-actions/job-actions.component';
import { JobBoxComponent } from './job-box/job-box.component';
import { JobDetailsComponent } from './job-details/job-details.component';

import { PipesModule } from '../../pipes';
import { ComponentsModule } from '../../common';

import { RecentCommitsComponent } from './recent-commits/recent-commits.component';
import { JobDetailsBoxComponent } from './job-details/job-details-box/job-details-box.component';
import { CommitDetailsBoxComponent } from './job-details/commit-details-box/commit-details-box.component';
import { TemplateDetailsBoxComponent } from './job-details/template-details-box/template-details-box.component';
import { FixturesDetailsBoxComponent } from './job-details/fixtures-details-box/fixtures-details-box.component';
import { JobsHistoryComponent } from './jobs-history/jobs-history.component';
import { JobsSummaryBoxComponent } from './jobs-history/jobs-summary-box/jobs-summary-box.component';
import { JobsDetailsBoxComponent } from './jobs-history/jobs-details-box/jobs-details-box.component';
import { JobsOverviewComponent } from './jobs-overview/jobs-overview.component';
import { JobFixturesBoxComponent } from './job-details/job-fixtures-box/job-fixtures-box.component';
import { JobStepSummaryComponent } from './job-step-summary/job-step-summary.component';
import { TimelineComponent } from './timeline/timeline.component';
import { RevisionComponent } from './revision/revision.component';
import { CommitsOverviewComponent } from './commits-overview/commits-overview.component';
import { BranchesOverviewComponent } from './branches-overview/branches-overview.component';
import { JobsTimelineComponent } from './jobs-timeline/jobs-timeline.component';
import { TimelineComponentsModule } from './timeline-components.module';

export const routes = [
    { path: '', component: TimelineComponent, terminal: true },
    { path: 'commits/:revisionId', component: RevisionComponent, terminal: true },
    { path: 'jobs/:id', component: JobDetailsComponent, terminal: true }
];

@NgModule({
    declarations: [
        BranchesOverviewComponent,
        JobsTimelineComponent,
        CommitBoxComponent,
        JobActionsComponent,
        JobBoxComponent,
        JobDetailsComponent,
        RecentCommitsComponent,
        JobDetailsBoxComponent,
        CommitDetailsBoxComponent,
        TemplateDetailsBoxComponent,
        FixturesDetailsBoxComponent,
        JobsHistoryComponent,
        JobsSummaryBoxComponent,
        JobsDetailsBoxComponent,
        JobsOverviewComponent,
        JobFixturesBoxComponent,
        JobStepSummaryComponent,
        RevisionComponent,
        CommitsOverviewComponent,
        TimelineComponent,
    ],
    imports: [
        CommonModule,
        PipesModule,
        ComponentsModule,
        FormsModule,
        TimelineComponentsModule,
        RouterModule.forChild(decorateRouteDefs(routes)),
    ]
})
export default class TimelineModule {

}

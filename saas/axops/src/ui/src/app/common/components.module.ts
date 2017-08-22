import { NgModule } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule, ReactiveFormsModule } from '@angular/forms';
import { RouterModule } from '@angular/router';
import { nvD3 } from 'ng2-nvd3';

import { PipesModule } from '../pipes';

import { GuiComponentsModule } from 'argo-ui-lib/src/components';

import { BranchesFiltersComponent } from './branches-filters/branches-filters.component';
import { InputSelectComponent } from './input-select/input-select.component';
import { LabelsFiltersComponent } from './labels-filters/labels-filters.component';
import { LoaderComponent } from './loader/loader.component';
import { LoaderInBackgroundComponent } from './loader-in-background/loader-in-background.component';
import { LoaderListMockupComponent } from './loader-list-mockup/loader-list-mockup.component';
import { ModalComponent } from './modal/modal.component';
import { StatusIconDirective } from './status-icon/status-icon.directive';
import { NotificationComponent } from './notification/notification.component';
import { SearchInputComponent } from './search-input/search-input.component';

import { TemplatesFiltersComponent } from './templates-filters/templates-filters.component';
import { ZipViewerComponent } from './zip-viewer/zip-viewer.component';
import { HighlightSubstringDirective } from './highlight-substring/highlight-substring.directive';
import { ModalTemplateComponent } from './modal/modal.template';
import { AvatarComponent } from './avatar/avatar.component';
import { ProgressBarComponent } from './progress-bar/progress-bar.component';
import { CommitPanelComponent } from './commit-panel/commit-panel.component';
import { CommitAuthorComponent } from './commit-author/commit-author.component';
import { CommitDescriptionComponent } from './commit-description/commit-description.component';
import { MultipleServiceLaunchPanelComponent } from './multiple-service-launch-panel/multiple-service-launch-panel.component';
import { LaunchPanelService } from './multiple-service-launch-panel/launch-panel.service';
import { BranchesPanelComponent } from './branches-panel/branches-panel.component';
import { BranchSelectorPanelComponent } from './branch-selector-panel/branch-selector-panel.component';
import { CommitSelectorPanelComponent } from './commit-selector-panel/commit-selector-panel.component';
import { WorkflowSubtreeComponent } from './workflow-tree/workflow-subtree.component';
import { WorkflowTreeNodeComponent } from './workflow-tree/workflow-tree-node.component';
import { WorkflowTreeComponent } from './workflow-tree/workflow-tree.component';
import { YamlViewerComponent } from './yaml-viewer/yaml-viewer.component';
import { TemplateViewerComponent } from './template-viewer/template-viewer.component';
import { ArtifactTagManagementComponent } from './artifact-tag-management/artifact-tag-management.component';
import { SysConsoleComponent } from './sys-console/sys-console.component';
import { JobStatusesComponent } from './job-statuses/job-statuses.component';
import { PieChartComponent } from './pie-chart/pie-chart.component';
import { PaginationComponent } from './pagination/pagination.component';
import { GlobalSearchInputComponent } from './global-search-input/global-search-input.component';
import { TimerangePaginationComponent } from './pagination/timerange-pagination.component';
import { VolumesListComponent } from './volumes-list/volumes-list.component';
import { IfAuthenticatedDirective } from './if-authenticated/if-authenticated.directive';
import { IfAnonymousDirective } from './if-authenticated/if-anonymous.directive';
import { AccessMarkComponent } from './access-mark/access-mark.component';
import { UsersSelectorPanelComponent } from './users-selector-panel/users-selector-panel.component';
import { SlackChannelsPanelComponent } from './slack-channels-panel/slack-channels-panel.component';
import { IfUserIsInGroupDirective } from './if-user-is-in-group/if-user-is-in-group.directive';
import { SelectSearchComponent } from './select-search/select-search.component';
import { JiraIssueCreatorPanelComponent } from './jira-issue-creator-panel/jira-issue-creator-panel.component';
import { JiraIssueTypeComponent } from './jira-issue-type/jira-issue-type.component';
import { JiraStatusComponent } from './jira-status/jira-status.component';
import { JiraIssuesListComponent } from './jira-issues-list/jira-issues-list.component';
import { JiraIssuesPanelComponent } from './jira-issues-panel/jira-issues-panel.component';
import { AttributesPanelComponent } from './attributes-panel/attributes-panel.component';
import { AutocompleteOffDirective } from './autocomplete-off/autocomplete-off.directive';
import { RedirectPanelComponent } from './redirect-panel/redirect-panel.component';
import { ApplicationStatusComponent } from './application-status/application-status.component';
import { ArtifactsComponent } from './artifacts/artifacts.component';

let components: any[] = [
    AccessMarkComponent,
    AvatarComponent,
    BranchesFiltersComponent,
    InputSelectComponent,
    HighlightSubstringDirective,
    LabelsFiltersComponent,
    LoaderComponent,
    LoaderInBackgroundComponent,
    LoaderListMockupComponent,
    SysConsoleComponent,
    ModalComponent,
    StatusIconDirective,
    ProgressBarComponent,
    NotificationComponent,
    SearchInputComponent,
    TemplatesFiltersComponent,
    ZipViewerComponent,
    ModalTemplateComponent,
    CommitPanelComponent,
    CommitAuthorComponent,
    CommitDescriptionComponent,
    MultipleServiceLaunchPanelComponent,
    BranchesPanelComponent,
    BranchSelectorPanelComponent,
    CommitSelectorPanelComponent,
    WorkflowSubtreeComponent,
    WorkflowTreeNodeComponent,
    WorkflowTreeComponent,
    YamlViewerComponent,
    TemplateViewerComponent,
    ArtifactTagManagementComponent,
    JobStatusesComponent,
    PieChartComponent,
    GlobalSearchInputComponent,
    PaginationComponent,
    TimerangePaginationComponent,
    IfAuthenticatedDirective,
    IfAnonymousDirective,
    nvD3,
    VolumesListComponent,
    UsersSelectorPanelComponent,
    SlackChannelsPanelComponent,
    IfUserIsInGroupDirective,
    SelectSearchComponent,
    JiraIssueCreatorPanelComponent,
    JiraIssueTypeComponent,
    JiraStatusComponent,
    JiraIssuesListComponent,
    JiraIssuesPanelComponent,
    AttributesPanelComponent,
    AutocompleteOffDirective,
    RedirectPanelComponent,
    ApplicationStatusComponent,
    ArtifactsComponent,
];

@NgModule({
    declarations: components,
    exports: components.concat(GuiComponentsModule),
    entryComponents: [ModalTemplateComponent],
    providers: [
        {
            provide: LaunchPanelService,
            useFactory: () => LaunchPanelService.create(),
        },
    ],
    imports: [
        GuiComponentsModule,
        CommonModule,
        FormsModule,
        ReactiveFormsModule,
        PipesModule,
        RouterModule,
    ],
})
export class ComponentsModule {
}

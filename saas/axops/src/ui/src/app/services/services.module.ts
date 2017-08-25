import { Injector, NgModule } from '@angular/core';

import { Http, HttpModule, RequestOptions, XHRBackend } from '@angular/http';
import { Router } from '@angular/router';
import { ToasterService } from 'angular2-toaster/angular2-toaster';
import { TranslateLoader, TranslateStaticLoader } from 'ng2-translate/ng2-translate';
import { HttpInterceptor } from '../common/httpInterceptors/HttpInterceptors';

import { AuthorizationService, HasNoSession, UserAccessControl } from './auth.service';
import { AuthenticationService } from './authentication.service';
import { BranchService } from './branch.service';
import { CommitsService } from './commit.service';
import { CookieService } from './cookie.service';
import { EventsService } from './events.service';
import { GroupService } from './group.service';
import { HostService } from './host.service';
import { ImageService } from './image.service';
import { JiraService } from './jira.service';
import { LabelService } from './label.service';
import { LoaderService } from './loader.service';
import { LoaderInBackgroundService } from './loader-in-background.service';
import { ModalService } from './modal.service';
import { NotificationService } from './notification.service';
import { PerfDataService } from './perf-data.service';
import { PerformanceService } from './performance.service';
import { PoliciesService } from './policies.service';
import { RepoService } from './repo.service';
import { TaskService } from './task.service';
import { StatusService } from './status.service';
import { TemplateService } from './template.service';
import { ToolService } from './tool.service';
import { UsageService } from './usage.service';
import { UsersService } from './users.service';
import { FixtureService } from './fixture.service';
import { CustomViewService } from './custom-views.service';
import { SystemService } from './system.service';
import { ConfigsService } from './configuration.service';
import { HttpService } from './http.service';
import { ViewPreferencesService } from './view-preferences.service';
import { ArtifactsService } from './artifacts.service';
import { ApplicationsService } from './applications.service';
import { ProjectService } from './project.service';
import { DeploymentsService } from './deployments.service';
import { RetentionPolicyService } from './retention-policy.service';
import { ContentService } from './content.service';
import { VolumesService } from './volumes.service';
import { GlobalSearchService } from './global-search.service';
import { SecretService } from './secret.service';
import { PlaygroundInfoService } from './playground-info.service';
import { TrackingService } from './tracking.service';
import { SlackService } from './slack.service';
import { SystemRequestService } from './system-request.service';

@NgModule({
    providers: [
        AuthorizationService,
        AuthenticationService,
        BranchService,
        CommitsService,
        CookieService,
        EventsService,
        GroupService,
        HostService,
        ImageService,
        JiraService,
        LabelService,
        LoaderService,
        LoaderInBackgroundService,
        ModalService,
        NotificationService,
        PerfDataService,
        PerformanceService,
        PoliciesService,
        RepoService,
        TaskService,
        SlackService,
        StatusService,
        SystemRequestService,
        TemplateService,
        ToolService,
        UsageService,
        UsersService,
        FixtureService,
        ConfigsService,
        CustomViewService,
        SystemService,
        HttpService,
        HasNoSession,
        UserAccessControl,
        ViewPreferencesService,
        ArtifactsService,
        ProjectService,
        ApplicationsService,
        DeploymentsService,
        RetentionPolicyService,
        VolumesService,
        GlobalSearchService,
        SecretService,
        TrackingService,
        PlaygroundInfoService,
        {
            provide: ContentService,
            useFactory: (http: Http, systemService: SystemService) => ENV === 'production' ?
                new ContentService(http, systemService.getVersion().toPromise().then(
                    versionInfo => {
                        let version = versionInfo.version.split('-')[0];
                        return `https://s3-us-west-1.amazonaws.com/ax-public/docs/${version}`;
                     })) :
                new ContentService(http, Promise.resolve('/assets/docs')),
            deps: [Http, SystemService],
        },
        {
            provide: TranslateLoader,
            useFactory: (http: Http) => new TranslateStaticLoader(http, 'assets/i18n', '.json'),
            deps: [Http]
        },
        {
            provide: Http,
            useFactory: (backend: XHRBackend,
                         defaultOptions: RequestOptions,
                         router: Router,
                         loaderService: LoaderService,
                         toasterService: ToasterService,
                         cookieService: CookieService,
                         injector: Injector) => new HttpInterceptor(backend,
                defaultOptions,
                router,
                loaderService,
                toasterService,
                cookieService,
                injector),
            deps: [XHRBackend,
                RequestOptions,
                Router,
                LoaderService,
                ToasterService,
                CookieService,
                Injector]
        },
    ],
    imports: [
        HttpModule
    ]
})
export class ServicesModule {

}

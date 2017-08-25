import * as moment from 'moment';
import { Component, ViewChild, OnInit, OnDestroy } from '@angular/core';
import { RouterOutlet, Router, ActivatedRoute } from '@angular/router';
import { Subject, Subscription, Observable } from 'rxjs';
import { FormControl, FormGroup, Validators } from '@angular/forms';

import { DropdownMenuSettings, SlidingPanelService, DateRange } from 'argo-ui-lib/src/components';
import { GlobalSearchSetting, LaunchPanelService, MultipleServiceLaunchPanelComponent, JiraIssueCreatorPanelComponent, JiraIssuesPanelComponent } from '../../common';
import {
    JiraService,
    NotificationService,
    PlaygroundInfoService,
    PlaygroundTaskInfo,
    RepoService,
    SecretService,
    ViewPreferencesService,
    AuthenticationService,
} from '../../services';
import { Task, Application } from '../../model';

export interface LayoutSettings {
    hasExtendedBg?: boolean;
    hiddenToolbar?: boolean;
    pageTitle?: string;
    pageTitleIcon?: string;
    breadcrumb?: { title: string, routerLink?: any[] }[];
    searchString?: string;
    globalAddAction?: () => void;
    globalAddActionMenu?: DropdownMenuSettings;
    branchNavPanelUrl?: string;
    globalSearch?: Subject<GlobalSearchSetting>;
    layoutDateRange?: {
        data: DateRange;
        onApplySelection: (any) => void;
        isAllDates?: boolean;
    };
    toolbarFilters?: {
        data: {
            name: string;
            value: string;
            icon?: {
                className?: string;
                color?: string; // 'success' | 'fail' | 'running' | 'queued';
            };
            hasSeparator?: boolean;
        }[];
        model: string[];
        onChange: (any) => void;
    };
    hasTabs?: boolean;
    customStickyPanelHeight?: number;
}

export interface HasLayoutSettings {
    layoutSettings: LayoutSettings;
}

@Component({
    selector: 'ax-layout',
    templateUrl: './layout.html',
    styles: [ require('./layout.scss') ],
})
export class LayoutComponent implements OnInit, OnDestroy {
    @ViewChild(RouterOutlet)
    public routerOutlet: RouterOutlet;
    @ViewChild(MultipleServiceLaunchPanelComponent)
    public multipleServiceLaunchPanel: MultipleServiceLaunchPanelComponent;
    @ViewChild(JiraIssueCreatorPanelComponent)
    public jiraIssueCreatorPanelComponent: JiraIssueCreatorPanelComponent;
    @ViewChild(JiraIssuesPanelComponent)
    public jiraIssuesPanelComponent: JiraIssuesPanelComponent;
    public globalSearch: GlobalSearchSetting;
    public hiddenScrollbar: boolean;
    public openedPanelOffCanvas: boolean;
    public tutorialVisible: boolean;
    public encryptionToolVisible: boolean = false;
    public encryptedSecret: string = '';
    public repos: string[] = [];
    public reposLoaded: boolean;
    public playgroundTask: PlaygroundTaskInfo = null;
    public showNotificationsCenter: boolean;
    public mostRecentEventTime: moment.Moment = moment.unix(0);
    public mostRecentNotificationsViewTime: moment.Moment = moment.unix(0);
    public encryptForm: FormGroup;
    public encryptFormSubmitted: boolean;
    public openedNav: boolean;
    public branchNavPanelOpened = false;

    private subscriptions: Subscription[] = [];

    constructor(
            private router: Router,
            private route: ActivatedRoute,
            private launchPanelService: LaunchPanelService,
            private slidingPanelService: SlidingPanelService,
            private jiraService: JiraService,
            private secretService: SecretService,
            private repoService: RepoService,
            private playgroundInfoService: PlaygroundInfoService,
            private notificationService: NotificationService,
            private viewPreferencesService: ViewPreferencesService,
            private authenticationService: AuthenticationService) {

        this.encryptForm = new FormGroup({
            repo: new FormControl('', Validators.required),
            secret: new FormControl('', Validators.required),
        });



        
        this.subscriptions.push(this.slidingPanelService.panelOpened.subscribe(
            isHidden => setTimeout(() => this.hiddenScrollbar = isHidden)));

        this.subscriptions.push(this.slidingPanelService.panelOffCanvasOpened.subscribe(
            openedPanelOffCanvas => setTimeout(() => this.openedPanelOffCanvas = openedPanelOffCanvas),
        ));

        this.subscriptions.push(route.queryParams.subscribe(async queryParams => {
            this.tutorialVisible = queryParams['tutorial'] === 'true';
            let encryptionToolVisible = queryParams['encryptionTool'] === 'true';
            if (this.encryptionToolVisible !== encryptionToolVisible) {
                this.encryptionToolVisible = encryptionToolVisible;
                this.encryptedSecret = '';
                if (this.encryptionToolVisible) {
                    this.encryptForm.reset();
                    this.encryptFormSubmitted = false;
                    this.reposLoaded = false;
                    let res = await this.repoService.getReposAsync(true).toPromise();
                    this.repos = res.data;
                    this.reposLoaded = true;
                }
            }
        }));

        this.subscriptions.push(playgroundInfoService.getPlaygroundTaskInfo().subscribe(info => {
            this.playgroundTask = info;
        }));

        this.authenticationService.getCurrentUser().then(user => {
            this.subscriptions.push(Observable.merge(
                this.notificationService.getEventsStream(user.username),
                    Observable.fromPromise(this.notificationService.getEvents({ limit: 1, recipient: user.username })).filter(events => events.length > 0).map(events => events[0])
                ).subscribe(event => {
                    this.mostRecentEventTime = moment(event.timestamp / 1000);
                }));
        });

        this.subscriptions.push(Observable.merge(
            Observable.fromPromise(viewPreferencesService.getViewPreferences()),
            viewPreferencesService.onPreferencesUpdated.asObservable(),
        ).subscribe(viewPreferences => {
            this.mostRecentNotificationsViewTime = moment.unix(viewPreferences.mostRecentNotificationsViewTime || 0);
        }));

        this.subscriptions.push(
            this.jiraService.showJiraIssueCreatorPanel.subscribe((value: {
                isVisible: boolean,
                associateWith: 'service' | 'application' | 'deployment',
                itemId: string,
                name: string,
                itemUrl: string}) => {
            this.jiraIssueCreatorPanelComponent.isVisibleJiraProjectSelectorPanel = value.isVisible;
            this.jiraIssueCreatorPanelComponent.serviceId = value.itemId;
            this.jiraIssueCreatorPanelComponent.itemUrl = value.itemUrl;
            this.jiraIssueCreatorPanelComponent.name = value.name;
            this.jiraIssueCreatorPanelComponent.associateWith = value.associateWith;
        }));

        this.subscriptions.push(
            this.jiraService.showJiraIssuesListPanel.subscribe((value: {
                isVisible: boolean,
                associateWith: 'service' | 'application' | 'deployment',
                item: Task | Application,
                itemUrl: string}) => {
            this.jiraIssuesPanelComponent.isVisibleJiraIssuesPanel = value.isVisible;
            this.jiraIssuesPanelComponent.itemUrl = value.itemUrl;
            this.jiraIssuesPanelComponent.item = value.item;
            this.jiraIssuesPanelComponent.associateWith = value.associateWith;
        }));
    }

    public get layoutSettings(): LayoutSettings {
        let component: any = this.routerOutlet.isActivated ? this.routerOutlet.component : null;
        return component ? component.layoutSettings || {} : {};
    }

    public ngOnInit() {
        this.launchPanelService.initPanel(this.multipleServiceLaunchPanel);
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
    }

    public getReposMenu(): DropdownMenuSettings {
        let items = this.repos.map(repo => ({
            title: repo,
            action: async () => {
                this.encryptForm.controls['repo'].setValue(repo);
            },
            iconName: 'ax-icon-branch',
        }));
        return new DropdownMenuSettings(items);
    }

    public closeEncryptionTool() {
        this.router.navigate([], { queryParams: { encryptionTool: 'false' } });
    }

    public openBranchNavPanel() {
        this.branchNavPanelOpened = true;
    }

    public closeBranchNavPanel() {
        this.branchNavPanelOpened = false;
    }

    public toggleNotificationsCenter(status: boolean) {
        this.showNotificationsCenter = status;
        if (this.showNotificationsCenter) {
            this.viewPreferencesService.updateViewPreferences(viewPreferences => {
                viewPreferences.mostRecentNotificationsViewTime = moment.utc().unix();
            });
        }
    }

    public toggleNav(status?: boolean) {
        this.openedNav = typeof status !== 'undefined' ? status : !this.openedNav;
    }

    public get animateNotificationIcon(): boolean {
        return this.mostRecentEventTime.isAfter(this.mostRecentNotificationsViewTime);
    }

    public async onEncryptFormSubmit(form: FormGroup) {
        this.encryptFormSubmitted = true;
        if (form.valid) {
            this.encryptedSecret = await this.secretService.encrypt(form.value.secret, form.value.repo);
        }
    }
}

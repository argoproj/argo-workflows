import * as moment from 'moment';
import { Component, ViewChild, OnInit, OnDestroy } from '@angular/core';
import { RouterOutlet, Router, NavigationEnd, ActivatedRoute } from '@angular/router';
import { Subject, Subscription, Observable } from 'rxjs';

import { DropdownMenuSettings, SlidingPanelService, DateRange } from 'argo-ui-lib/src/components';
import { GlobalSearchSetting, LaunchPanelService, MultipleServiceLaunchPanelComponent, JiraIssueCreatorPanelComponent, JiraIssuesPanelComponent } from '../../common';
import {
    JiraService,
    NotificationService,
    PlaygroundInfoService,
    PlaygroundTaskInfo,
    ViewPreferencesService,
    AuthenticationService,
    SharedService
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
    public layoutSettings: LayoutSettings = {};
    public globalSearch: GlobalSearchSetting;
    public hiddenScrollbar: boolean;
    public openedPanelOffCanvas: boolean;
    public tutorialVisible: boolean;
    public repos: string[] = [];
    public reposLoaded: boolean;
    public playgroundTask: PlaygroundTaskInfo = null;
    public showNotificationsCenter: boolean;
    public mostRecentEventTime: moment.Moment = moment.unix(0);
    public mostRecentNotificationsViewTime: moment.Moment = moment.unix(0);
    public openedNav: boolean;
    public branchNavPanelOpened = false;

    private subscriptions: Subscription[] = [];

    constructor(
            private router: Router,
            private route: ActivatedRoute,
            private launchPanelService: LaunchPanelService,
            private slidingPanelService: SlidingPanelService,
            private jiraService: JiraService,
            private playgroundInfoService: PlaygroundInfoService,
            private notificationService: NotificationService,
            private viewPreferencesService: ViewPreferencesService,
            private authenticationService: AuthenticationService,
            private sharedService: SharedService) {

        this.subscriptions.push(router.events.subscribe(event => {
            if (event instanceof NavigationEnd) {
                let component: any = this.routerOutlet.component;
                this.layoutSettings = component ? component.layoutSettings || {} : {};
            }
        }));

        this.subscriptions.push(this.slidingPanelService.panelOpened.subscribe(
            isHidden => setTimeout(() => this.hiddenScrollbar = isHidden)));

        this.subscriptions.push(this.slidingPanelService.panelOffCanvasOpened.subscribe(
            openedPanelOffCanvas => setTimeout(() => this.openedPanelOffCanvas = openedPanelOffCanvas),
        ));

        this.subscriptions.push(route.queryParams.subscribe(async queryParams => {
            this.tutorialVisible = queryParams['tutorial'] === 'true';
        }));

        this.subscriptions.push(playgroundInfoService.getPlaygroundTaskInfo().subscribe(info => {
            this.playgroundTask = info;
        }));

        this.subscriptions.push(this.sharedService.updateSource.subscribe((layout) => {
            this.layoutSettings = layout;
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

    public ngOnInit() {
        this.launchPanelService.initPanel(this.multipleServiceLaunchPanel);
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
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
}

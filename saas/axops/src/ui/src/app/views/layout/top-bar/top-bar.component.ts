import { Component, Input, OnChanges, SimpleChange, EventEmitter, Output, OnInit } from '@angular/core';
import { BehaviorSubject } from 'rxjs/BehaviorSubject';

import { LayoutSettings } from '../layout.component';
import { ContentService, SystemService, GlobalSearchService, AuthenticationService } from '../../../services';
import { User } from '../../../model';
import { GLOBAL_SEARCH_TABS, GlobalSearchSetting } from '../../../common';

@Component({
    selector: 'ax-top-bar',
    templateUrl: './top-bar.html',
    styles: [ require('./top-bar.component.scss') ],
})
export class TopBarComponent implements OnChanges, OnInit {

    public isGlobalSearchVisible: boolean;
    public releaseNotesUrl: string = '';
    public isPlayground = false;
    public userObj: User;

    @Input()
    public settings: LayoutSettings;

    @Input()
    public animateNotificationIcon: boolean;

    @Input()
    public openedNotificationsCenter: boolean;

    @Output()
    public onOpenNotificationsCenter: EventEmitter<any> = new EventEmitter();

    constructor(
        private contentService: ContentService,
        private systemService: SystemService,
        private globalSearchService: GlobalSearchService,
        private authenticationService: AuthenticationService) {}

    public ngOnInit() {
        // Create an instance of user object. The authentication service returns a json converted object.
        // not changing authentication service to avoid any impact in session management
        this.userObj = new User(this.authenticationService.getUser());

        this.contentService.getDocUrls().then(urls => this.releaseNotesUrl = urls.releaseNotesUrl);
        this.systemService.isPlayground().then(isPlayground => this.isPlayground = isPlayground);

        this.globalSearchService.toggleGlobalSearch.subscribe(res => {
            this.isGlobalSearchVisible = res;
        });
    }

    public ngOnChanges(changes: { [propertyName: string]: SimpleChange }) {
        if (changes && changes.hasOwnProperty('settings')) {
            if (this.settings.pageTitle) {
                document.title = 'Argo | ' + this.settings.pageTitle;
            } else {
                document.title = 'Argo';
            }

            // if there is no local or global search add default search same as /jobs page
            if (!this.settings.hasOwnProperty ('globalSearch')) {
                this.settings.globalSearch = new BehaviorSubject<GlobalSearchSetting>({
                    suppressBackRoute: false,
                    keepOpen: false,
                    searchCategory: GLOBAL_SEARCH_TABS.JOBS.name,
                    hideSearchHistoryAndSuggestions: true,
                });
            }
        }
    }

    public openNotificationsCenter() {
        this.onOpenNotificationsCenter.emit(null);
    }

    public doSignOut() {
        this.authenticationService.logout();
    }
}

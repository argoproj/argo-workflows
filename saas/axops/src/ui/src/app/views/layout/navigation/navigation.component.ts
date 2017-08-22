import * as _ from 'lodash';
import { Subscription } from 'rxjs';
import { Component, Output, EventEmitter, Input, OnInit, OnDestroy } from '@angular/core';
import { Router } from '@angular/router';

import { ViewPreferencesService, ToolService, SystemService, RepoService } from '../../../services';
import { Branch, ViewPreferences, Repository } from '../../../model';
import { SortOperations } from '../../../common/sortOperations/sortOperations';

@Component({
    selector: 'ax-navigation',
    templateUrl: './navigation.html',
    styles: [ require('./navigation.component.scss') ],
})
export class NavigationComponent implements OnInit, OnDestroy {

    public repositories: Repository[];
    public loading = false;

    @Input()
    public blocked = false;

    @Input()
    public show: boolean;

    @Input()
    public branchNavPanelUrl: boolean;

    @Input()
    public branchNavPanelOpened: boolean;

    @Output()
    public onToggleNav: EventEmitter<any> = new EventEmitter();

    @Output()
    public onCloseNavPanel: EventEmitter<any> = new EventEmitter();

    public showFavoriteBranches: boolean;

    private favouriteBranches: Branch[];
    private subscriptions: Subscription[] = [];

    private version: string;

    constructor(private viewPreferencesService: ViewPreferencesService,
                private router: Router,
                private repoService: RepoService,
                private systemService: SystemService,
                private toolService: ToolService) {
    }

    public ngOnInit() {
        this.viewPreferencesService.getViewPreferences().then((v: ViewPreferences) => {
            this.favouriteBranches = v.favouriteBranches;
            this.showFavoriteBranches = v.filterState.branches === 'my';
        });

        this.systemService.getVersion().toPromise().then((info) => {
            this.version = info.version.split('-')[ 0 ];
        });
        this.loadRepos();
        this.subscriptions.push(this.toolService.onToolsChanged.subscribe(() => this.loadRepos()));
    }

    public loadRepos() {
        this.loading = true;
        this.repoService.getReposAsync(true).subscribe(
            res => {
                this.repositories = SortOperations.sortBy(_.map(res.data, (repo: string) => {
                    let repoSplit = repo.split('/');

                    return {
                        name: repoSplit[ repoSplit.length - 1 ],
                        url: repo,
                    };
                }), 'name', true);

                this.loading = false;
            },
        );
    }

    public ngOnDestroy() {
        this.subscriptions.forEach(subscription => subscription.unsubscribe());
        this.subscriptions = [];
    }

    public subNavAction(url: string, params?: any) {
        this.router.navigate([ url, params ]);
    }

    public onClosePanel() {
        this.onCloseNavPanel.emit({});
    }

    public close() {
        this.onToggleNav.emit(false);
    }

    public open() {
        this.onToggleNav.emit(true);
    }

    public toggleFavoriteBranchesEnabled() {
        this.viewPreferencesService.updateViewPreferences(preferences => {
            this.showFavoriteBranches = !this.showFavoriteBranches;
            preferences.filterState.branches = this.showFavoriteBranches ? 'my' : 'all';
        });
    }
}

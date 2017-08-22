import { Component, OnInit } from '@angular/core';
import { SystemService, ContentService, DocUrls } from '../../services';
import { VersionInfo } from '../../model';

@Component({
    selector: 'ax-help',
    templateUrl: './help.component.html',
    styles: [ require('./help.component.scss') ],
})
export class HelpComponent implements OnInit {

    public versionInfo: VersionInfo;
    public docUrls: DocUrls;

    constructor(private systemService: SystemService, private contentService: ContentService) {
    }

    public ngOnInit() {
        this.systemService.getVersion().subscribe(versionInfo => this.versionInfo = versionInfo);
        this.contentService.getDocUrls().then(urls => this.docUrls = urls);
    }
}

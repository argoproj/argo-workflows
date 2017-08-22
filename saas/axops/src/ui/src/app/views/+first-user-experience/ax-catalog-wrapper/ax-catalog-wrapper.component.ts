import { Component, OnInit, ViewChild } from '@angular/core';
import { AuthorizationService } from '../../../services';
import { LaunchPanelService } from '../../../common/multiple-service-launch-panel/launch-panel.service';
import { MultipleServiceLaunchPanelComponent } from '../../../common/multiple-service-launch-panel/multiple-service-launch-panel.component';

@Component({
    selector: 'ax-catalog-wrapper',
    templateUrl: './ax-catalog-wrapper.html',
    styles: [ require('./ax-catalog-wrapper.scss'), require('../first-user-experience.scss') ],
})
export class AxCatalogWrapperComponent implements OnInit {
    @ViewChild(MultipleServiceLaunchPanelComponent)
    multipleServiceLaunchPanel: MultipleServiceLaunchPanelComponent;

    constructor(private authorizationService: AuthorizationService, private launchPanelService: LaunchPanelService) {}

    public completeIntroduction() {
        this.authorizationService.completeIntroduction();
    }

    public ngOnInit() {
        this.launchPanelService.initPanel(this.multipleServiceLaunchPanel);
    }
}

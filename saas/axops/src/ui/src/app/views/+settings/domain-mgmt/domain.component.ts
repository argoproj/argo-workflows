import { Component, OnInit } from '@angular/core';

import { LayoutSettings, HasLayoutSettings } from '../../layout/layout.component';
import { ToolService, ModalService } from '../../../services';
import { NotificationsService } from 'argo-ui-lib/src/components';
import { Route53Config } from '../../../model';

@Component({
    selector: 'ax-domain-mgmt',
    templateUrl: './domain.component.html',
})

export class DomainManagementComponent implements LayoutSettings, HasLayoutSettings, OnInit {
    private domainConfig: Route53Config;
    private selectedDomains: any[] = [];
    private availableDomains: any[] = [];

    constructor(private toolService: ToolService,
        private notificationsService: NotificationsService,
        private modalService: ModalService) {
    }

    ngOnInit() {
        this.toolService.getToolsAsync({ category: 'domain_management' }).subscribe((success) => {
            if (success && success.data && success.data.length > 0) {
                this.setDomainConfig(new Route53Config(success.data[0]));
            } else {
                this.toolService.postToDomainManagement().subscribe((response) => {
                    this.setDomainConfig(response);
                });
            }
        });

    }
    get layoutSettings(): LayoutSettings {
        return this;
    }

    get pageTitle(): string {
        return 'Domain Management';
    };

    public breadcrumb: { title: string, routerLink?: any[] }[] = [{
        title: `Settings`,
        routerLink: [`/app/settings/overview`],
    }, {
        title: `Domain Management`,
    }];

    public updateSelection() {
        let selectedItems = [];
        this.selectedDomains.forEach((item) => {
            selectedItems.push({ name: item });
        });

        this.domainConfig.domains = selectedItems;
        this.toolService.postToDomainManagement(this.domainConfig).subscribe((response) => {
            this.setDomainConfig(response);
            this.notificationsService.success(`Domains list updated successfuly.`);
        }, (error) => {
            this.notificationsService.error(`Unable to update domains.`);
        });
    }

    private setDomainConfig(data: Route53Config) {
        this.domainConfig = data;

        this.availableDomains = this.domainConfig.all_domains || [];

        let selected = [];
        if (this.domainConfig.domains) {
            selected = this.availableDomains.filter(domain => {
                return this.domainConfig.domains.findIndex(item => item.name === domain) > -1;
            });
        }
        this.selectedDomains = selected;
    }
}

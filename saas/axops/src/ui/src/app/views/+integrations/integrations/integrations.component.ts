import { Component, OnInit, TemplateRef, ViewChild } from '@angular/core';
import { ActivatedRoute } from '@angular/router';

import { ToolService } from '../../../services';
import { ITool, ToolCategory } from '../../../model';
import { HasLayoutSettings, LayoutSettings } from '../../layout';
import { REGISTRY_TYPES } from '../../../model';

interface ToolGroupInfo {
    name: string;
    type: string;
    category: string;
    tools: ITool[];
    panelTemplate: TemplateRef<any>;
    maxToolsCount?: number;
}

@Component({
    selector: 'ax-integrations',
    templateUrl: './integrations.html',
    styles: [ require('./integrations.scss') ]
})
export class IntegrationsComponent implements OnInit, LayoutSettings, HasLayoutSettings {
    public groupsInfo: ToolGroupInfo[] = [];
    public pageTitle: string;
    public header: string;
    public selectedToolGroup: ToolGroupInfo;
    public selectedTool: ITool;
    public createNew: boolean = false;
    public registerTypes = REGISTRY_TYPES;

    @ViewChild('bitbucketTemplate')
    public bitbucketTemplate: TemplateRef<any>;
    @ViewChild('gitTemplate')
    public gitTemplate: TemplateRef<any>;
    @ViewChild('githubTemplate')
    public githubTemplate: TemplateRef<any>;
    @ViewChild('codecommitTemplate')
    public codecommitTemplate: TemplateRef<any>;
    @ViewChild('smtpTemplate')
    public smtpTemplate: TemplateRef<any>;
    @ViewChild('nexusTemplate')
    public nexusTemplate: TemplateRef<any>;
    @ViewChild('registryTemplate')
    public registryTemplate: TemplateRef<any>;
    @ViewChild('dockerhubRegistryTemplate')
    public dockerhubRegistryTemplate: TemplateRef<any>;
    @ViewChild('slackTemplate')
    public slackTemplate: TemplateRef<any>;
    @ViewChild('jiraTemplate')
    public jiraTemplate: TemplateRef<any>;
    @ViewChild('gitLabTemplate')
    public gitLabTemplate: TemplateRef<any>;

    private category: ToolCategory = 'scm';

    constructor(private route: ActivatedRoute, private toolService: ToolService) {}

    public ngOnInit() {
        this.route.params.subscribe(params => {
            this.category = params['type'] || 'scm';
            this.pageTitle = this.categoryInfo.pageTitle;
            this.header = this.categoryInfo.header;
            this.reloadTools().then(() => {
                this.selectToolGroup(this.groupsInfo[0]);
            });
        });
    }

    public get breadcrumb(): { title: string, routerLink?: any[] }[] {
        return [
            {
                title: `Integrations`,
                routerLink: [ `/app//integrations/overview` ],
            }, {
                title: this.categoryInfo.pageTitle,
            }
        ];
    };

    public onToolDeleted() {
        this.reloadTools().then(() => {
            this.selectToolGroup(this.selectedToolGroup ?
                this.groupsInfo.find(group => group.type === this.selectedToolGroup.type) : null);
            this.selectedTool = null;
        });
    }

    public onToolCreated(newTool: ITool) {
        this.reloadTools().then(() => {
            this.selectedToolGroup = this.selectedToolGroup ?
                    this.groupsInfo.find(group => group.type === this.selectedToolGroup.type) : null;
            if (this.selectedTool === null) {
                this.selectTool(this.selectedToolGroup ?
                    this.selectedToolGroup.tools.find(tool => tool.id === newTool.id) : null);
            } else {
                this.selectTool(null);
            }
        });
    }

    public selectToolGroup(group: ToolGroupInfo) {
        this.selectedToolGroup = group;
        this.selectedTool = null;
        this.createNew = group && group.tools && group.tools.length === 0;
    }

    public selectTool(tool: ITool) {
        this.selectedTool = tool;
        if (tool) {
            this.createNew = false;
        }
    }

    public setCreateNew(val: boolean) {
        this.createNew = val;
        if (val) {
            this.selectedTool = null;
        }
    }

    get canCreateMore(): boolean {
        return this.selectedToolGroup && (this.selectedToolGroup.maxToolsCount === undefined || this.selectedToolGroup.tools.length < this.selectedToolGroup.maxToolsCount);
    }

    get layoutSettings(): LayoutSettings {
        return this;
    }

    private reloadTools(): Promise<any> {
        return this.toolService.getToolsAsync({ category: this.category }).toPromise().then(res => {
            this.groupsInfo = this.categoryInfo.tools.map(group => {
                return Object.assign({}, group, {
                    tools: res.data
                        .filter(item => item.type === group.type)
                        .sort((first: ITool, second: ITool) => (first.url || '').localeCompare(second.url))
                });
            });
        });
    }

    private get categoryInfo() {
        switch (this.category) {
            case 'scm':
                return {
                    pageTitle: 'Source Control',
                    header: 'Connect your source control',
                    tools: this.getSourceControlTools(),
                };
            case 'notification':
                return {
                    pageTitle: 'Notifications',
                    header: 'Setup your notifications',
                    tools: this.getNotificationsTools(),
                };
            case 'artifact_management':
                return {
                    pageTitle: 'Artifact Management',
                    header: 'Connect artifact repositories',
                    tools: this.getArtifactsManagementTools(),
                };
            case 'registry':
                return {
                    pageTitle: 'Container Registry',
                    header: 'Container Registry',
                    tools: this.getContainerRegistryTools(),
                };
            case 'issue_management':
                return {
                    pageTitle: 'Issue Tracking',
                    header: 'Connect Issue Tracking System',
                    tools: this.getIssueTrackingTools(),
                };
        }
    }

    private getSourceControlTools() {
        return [{
            name: 'Bitbucket',
            type: 'bitbucket',
            icon: 'fa fa-bitbucket',
            category: 'scm',
            panelTemplate: this.bitbucketTemplate,
        }, {
            name: 'GitHub',
            type: 'github',
            icon: 'fa fa-github',
            category: 'scm',
            panelTemplate: this.githubTemplate,
        }, {
            name: 'Git',
            type: 'git',
            icon: 'ax-icon-git',
            category: 'scm',
            panelTemplate: this.gitTemplate,
        }, {
            name: 'CodeCommit',
            type: 'codecommit',
            icon: 'code-commit',
            category: 'scm',
            panelTemplate: this.codecommitTemplate,
        }, {
            name: 'GitLab',
            type: 'gitlab',
            icon: 'ax-icon-git',
            category: 'scm',
            panelTemplate: this.gitLabTemplate,
        }];
    }

    private getNotificationsTools() {
        return [{
            name: 'SMTP',
            icon: 'fa fa-bell',
            type: 'smtp',
            category: 'notification',
            panelTemplate: this.smtpTemplate,
        }, {
            name: 'Slack',
            icon: 'fa fa-slack',
            type: 'slack',
            category: 'notification',
            panelTemplate: this.slackTemplate,
        }];
    }

    private getArtifactsManagementTools() {
        return [{
            name: 'Nexus',
            icon: 'fa ax-icon-intlogo',
            type: 'nexus',
            category: 'artifact',
            panelTemplate: this.nexusTemplate,
        }];
    }

    private getContainerRegistryTools() {
        return [{
            name: 'Docker Hub',
            icon: 'fa ax-icon-docker',
            type: this.registerTypes.dockerhub,
            category: 'registry',
            panelTemplate: this.dockerhubRegistryTemplate,
            maxToolsCount: 1,
        }, {
            name: 'Private Registry',
            icon: 'fa ax-icon-docker',
            type: this.registerTypes.privateRegistry,
            category: 'registry',
            panelTemplate: this.registryTemplate,
        }];
    }

    private getIssueTrackingTools() {
        return [{
            name: 'Jira',
            icon: 'ax-icon-jira',
            type: 'jira',
            category: 'issue-tracking',
            panelTemplate: this.jiraTemplate,
            maxToolsCount: 1,
        }];
    }
}

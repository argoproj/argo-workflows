import * as _ from 'lodash';
import { Component, Output, EventEmitter } from '@angular/core';
import { Router } from '@angular/router';
import { FormGroup, FormControl, Validators } from '@angular/forms';
import { Subscription } from 'rxjs/Subscription';
import { Observable } from 'rxjs/Observable';

import {Artifact, Commit, Template, Task, Project, ProjectAction, Branch } from '../../model';
import { TemplateService, TaskService, CommitsService } from '../../services';
import { NotificationsService } from 'argo-ui-lib/src/components';
import { Session, HtmlForm, MULTIPLE_SERVICE_LAUNCH_PANEL_TABS } from './multiple-service-launch-panel.view-models';

@Component({
    selector: 'ax-multiple-service-launch-panel',
    templateUrl: './multiple-service-launch-panel.html',
    styles: [ require('./multiple-service-launch-panel.scss') ],
})
export class MultipleServiceLaunchPanelComponent {
    @Output() submitted: EventEmitter<any> = new EventEmitter();

    public templates: Template[] = [];
    public templatesToSubmit: Template[] = [];
    public allSelected: boolean = false;
    public selectedItems: number = 0;
    public templateLoader: boolean = false;
    public commit: Commit = new Commit();
    public isVisibleSelectServiceTemplatesPanel: boolean = false;
    public selectingScreen: boolean = true;
    public activeElementId: number = 0;
    public isSubmitClicked: boolean = false;
    public summaryErrorMessage: boolean = false;
    public allFormsHtml: HtmlForm[] = [];
    public selectedBranch: string;
    public selcetedRepo: string;
    public isBranchEditable: boolean;
    public search: string;
    public selectedTab: string = MULTIPLE_SERVICE_LAUNCH_PANEL_TABS.PARAMETERS;

    private session: Session = new Session();
    private allForms: FormGroup;
    private subscription: Subscription;
    private projectInfo: { project: Project, action: ProjectAction };
    private showChangeRepoBranchPanel: boolean = false;
    private resubmit: boolean = false;
    private task: Task;
    private artifacts: Artifact[];
    private withCommitOnly: boolean;
    private allowedYamlParameters: any[] = [];

    constructor(private templateService: TemplateService,
                private taskService: TaskService,
                private commitsService: CommitsService,
                private notificationsService: NotificationsService,
                private router: Router) {
    }

    openPanel(
            commit: Commit,
            options: Task | Template | {project: Project, action: ProjectAction},
            withCommitOnly: boolean,
            artifacts: Artifact[],
            resubmit: boolean) {
        this.selectedTab = MULTIPLE_SERVICE_LAUNCH_PANEL_TABS.PARAMETERS;
        this.commit = new Commit();
        this.artifacts = artifacts || [];
        this.isBranchEditable = artifacts !== undefined;

        if (this.allForms) {
            this.allForms.reset();
            this.allFormsHtml = [];
            this.allForms = new FormGroup({});
        } else {
            this.allForms = new FormGroup({});
        }

        this.subscription = this.allForms.valueChanges.subscribe(() => {
            if (this.isSubmitClicked) {
                this.summaryErrorMessage = this.allForms.invalid;
            }
        });

        this.task = null;
        this.projectInfo = null;
        this.resubmit = resubmit;
        this.selectingScreen = true;

        this.withCommitOnly = withCommitOnly;
        if (options) {
            if (options['project'] && options['action']) {
                this.projectInfo = <{ project: Project, action: ProjectAction }> options;
            } else if (options['template']) { // resubmit failed
                this.task = <Task> options;
            } else {
                this.templates = [ Object.assign({}, <Template> options, { selected: true }) ];
                this.isVisibleSelectServiceTemplatesPanel = true;
                this.next();
            }
            this.selectingScreen = false;
        } else {
            this.task = null;
            this.projectInfo = null;
            this.selectingScreen = true;
        }

        if (commit) {
            if (!commit.branch && (commit.branches || []).length === 0) {
                this.commitsService.getCommitByRevision(commit.revision).subscribe(res => this.loadTemplates(res));
            } else {
                this.loadTemplates(commit);
            }
        }
    }

    loadTemplates(commit: Commit) {
        this.commit = commit;
        this.selcetedRepo = commit.repo;
        this.selectedBranch = commit.branch || commit.branches[0];
        this.isVisibleSelectServiceTemplatesPanel = true;

        this.getTemplates(commit.repo);
    }

    selectBranch(branch: string) {
        this.selectedBranch = branch;
        this.getTemplates(this.commit.repo);
    }

    editRepoBranch() {
        this.showChangeRepoBranchPanel = true;
    }

    closePanel(event?) {
        this.session = new Session();
        this.templates = [];
        this.templatesToSubmit = [];
        this.selectingScreen = true;
        this.activeElementId = 0;
        this.allSelected = false;
        this.allForms.reset();
        this.allFormsHtml = [];
        this.isSubmitClicked = false;
        this.summaryErrorMessage = false;
        this.isVisibleSelectServiceTemplatesPanel = false;
        this.subscription.unsubscribe();
    }

    getTemplates(repo: string) {
        if (this.task) {
            let template = Object.assign({ selected: true }, this.task.template);
            this.templates = [ template ];
            this.templatesToSubmit = [ template ];
            this.prepareForms(this.templatesToSubmit, this.task.arguments);
        } else {
            this.templateLoader = true;
            this.templates = [];
            let params = {repo_branch: `${repo}_${this.selectedBranch}`};
            if (this.withCommitOnly) {
                params['commit'] = true;
            }
            this.templateService.getTemplatesAsync(params, false).subscribe(res => {
                this.templates = res.data || [];
                if (this.artifacts.length) {
                    this.templates = this.filterTemplatesByArtifact(this.templates, this.artifacts);
                }

                this.templateLoader = false;
                if (this.projectInfo) {
                    this.templates.forEach(template => {
                        if (this.projectInfo.action.template === template.name) {
                            let projectParams = this.projectInfo.action.parameters || {};
                            template.selected = true;
                            let templateInputParams = (template.inputs || {}).parameters || {};
                            for (let paramName of Object.keys(templateInputParams)) {
                                if (projectParams.hasOwnProperty(paramName)) {
                                    templateInputParams[paramName].default = projectParams[paramName];
                                }
                            }
                        }
                    });
                    this.next();
                }
            }, error => {
                this.templateLoader = false;
            });
        }
    }

    filterTemplatesByArtifact(templates: Template[], artifacts: Artifact[]): Template[] {
        let clonedTemplates = JSON.parse(JSON.stringify(templates));
        let filteredTemplates = [];
        this.allowedYamlParameters = [];

        // I have to go throught all artifacts and get all templates with matching parameters
        // the prefix of the default value is: "%%artifacts.workflow." or "%%artifacts.tag." and postfix "$ARTIFACT_NAME%%"
        artifacts.forEach((artifact: Artifact) => {
            this.allowedYamlParameters.push([`%%artifacts.workflow.`, `.${artifact.name}%%`, artifact]);

            let templatesWithMatchingParameters = [];
            templatesWithMatchingParameters = clonedTemplates.filter(template => {
                for (let property in template.inputs.parameters) {
                    if (template.inputs.parameters.hasOwnProperty(property)) {
                        if (template.inputs.parameters[property].hasOwnProperty('default')
                            && (template.inputs.parameters[property].default.indexOf('%%artifacts.tag.') !== -1
                              || template.inputs.parameters[property].default.indexOf(`.${artifact.name}%%`) !== -1)) {
                            this.allowedYamlParameters.push([`%%artifacts.tag.`, `.${artifact.name}%%`, artifact]);
                            return template;
                        }
                    }
                }
            });

            filteredTemplates = filteredTemplates.concat(templatesWithMatchingParameters);
        });

        // return unique list
        return filteredTemplates.filter((v, i, a) => a.indexOf(v) === i);
    }

    get projectTemplate(): Template {
        return this.templates.find(template => template.name === this.projectInfo.action.template);
    }

    selectAllTemplates() {
        this.allSelected = !this.allSelected;
        this.selectedItems = 0;

        this.templates.forEach(p => {
            p.selected = this.allSelected;

            if (this.allSelected) {
                this.selectedItems++;
            }
        });
    }

    selectTemplate(template) {
        this.allSelected = false;
        template.selected = !template.selected;

        template.selected ? this.selectedItems++ : this.selectedItems--;
    }

    selectElement(index: number) {
        this.activeElementId = index;
    }

    isAnyTemplateSelected() {
        return this.templates.find(item => {
            return item['selected'] === true;
        }) !== undefined;
    }

    next() {
        this.selectingScreen = false;
        this.templatesToSubmit = this.templates.filter(item => item.selected);

        this.prepareForms(this.templatesToSubmit);
    }

    submit() {
        this.isSubmitClicked = true;
        this.summaryErrorMessage = this.allForms.invalid;

        if (this.allForms.valid) {
            if (this.resubmit) {
                this.resubmitTask(this.task, this.resubmit);
            } else {
                let observableList: Observable<any>[] = [];

                this.templatesToSubmit.forEach((template, index) => {
                    observableList.push(this.taskService.launchTask({
                        template_id: this.templatesToSubmit[index].id,
                        arguments: this.listParameters(this.allForms.controls[index.toString()]['controls']),
                    }));
                });

                Observable.forkJoin(observableList).subscribe(success => {
                        this.notificationsService.success(
                            `Successfully created a ${success.length} jobs for Commit: ${this.commit.description}`);
                        this.submitted.emit(success);
                        if (success.length === 1) {
                            this.router.navigate(['/app/timeline/jobs', success[0].id]);
                        }
                    }, error => {
                        this.notificationsService.internalError();
                    }
                );
            }
            this.closePanel();
        }
    }

    get isActiveFormEmpty(): boolean {
        return this.allForms.controls[this.activeElementId] && Object.keys(this.allForms.controls[this.activeElementId]['controls']).length === 0;
    }

    listParameters(formControl: FormControl) {
        let parameters: any = {};
        _.forOwn(formControl, (value, key) => {
            parameters[`parameters.${key}`] = value.value;
        });

        return parameters;
    }

    public selectTab(tabName: string) {
        this.selectedTab = tabName;
    }

    public selectedCommitChanged(commit: Commit) {
        this.commit = commit;
        this.session.commit = commit.revision;
        this.session.repo = commit.repo;
        this.session.branch = commit.branch;
        if (this.allForms.controls[0]['controls']) {
            this.allForms.controls[0]['controls'].commit.setValue(commit.revision);
        }
    }

    public selectedBranchChanged(branch: Branch) {
        this.selcetedRepo = branch.repo;
        this.selectedBranch = branch.name;
        this.getTemplates(branch.repo);
    }


    private prepareForms(templates: Template[], resubmitFailedParameters?: any) {
        templates.forEach((template, index) => {
            let newForm = new FormGroup({});
            let list = {
                name: index,
                parameters: []
            };
            let templateInputParams = (template.inputs || {}).parameters || {};
            for (let property in templateInputParams) {
                if (templateInputParams.hasOwnProperty(property)) {
                    let required = true;
                    let val = null;
                    if (resubmitFailedParameters) { // resubmit failed
                        val = resubmitFailedParameters[property];
                    } else if (templateInputParams[property] && templateInputParams[property].hasOwnProperty('default')) {
                        val = this.setParamValue(templateInputParams[property]['default'], template);
                        required = false;
                    }
                    newForm.addControl(property, new FormControl(val, required ? Validators.required : null));
                    list.parameters.push({
                        name: property,
                        value: val
                    });
                }
            }

            this.allFormsHtml.push(list);
            this.allForms.addControl(index.toString(), newForm);
        });
    }

    // if default value for parameter starts and ends with %%, replace the value with corresponding commits parameter
    private setParamValue(parameterValue: string, template: Template) {
        let vTemp = parameterValue;
        this.session = {
            commit: this.commit.revision || template.revision,
            repo: this.commit.repo || template.repo,
            branch: this.commit.branch || template.branch
        };

        // this is required to GUI-1367 launch a workflow using the workflow instance's exported artifact
        if (this.isBranchEditable &&
            (parameterValue.indexOf('%%artifacts.tag.') !== -1 || parameterValue.indexOf('%%artifacts.workflow.') !== -1 )) {
            let artifact = this.allowedYamlParameters.find(item => {
                if (parameterValue.indexOf(item[0]) !== -1 && parameterValue.indexOf(item[1]) !== -1 ) {
                    return item[2];
                }
            })[2];
            return artifact ? `%%artifacts.workflow.${artifact.workflow_id}.${artifact.name}%%` : parameterValue;
        }

        parameterValue = parameterValue.replace(/%%/g, '');

        return this.session[parameterValue.substring(parameterValue.indexOf('.') + 1).toString()] || vTemp;
    }

    private resubmitTask(task, runPartial = false): any {
        if (runPartial) {
            task.parameters = this.listParameters(this.allForms.controls[0]['controls']);
            let newCommit = task.parameters['commit'];
            let newRepo = task.parameters['repo'];
            task.parameters['session.commit'] = newCommit;
            task.parameters['session.repo'] = newRepo;
            task['newcommit'] = {
                revision: newCommit,
                repo: newRepo,
            };
        }
        this.taskService.launchTask(task, runPartial).subscribe(newTask => {
            this.notificationsService.success(`The job ${newTask.template.name} has been started.`);
            return true;
        });
    }
}

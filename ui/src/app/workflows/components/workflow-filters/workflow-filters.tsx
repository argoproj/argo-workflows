import * as React from 'react';
import * as models from '../../../../models';
import {CheckboxFilter} from '../../../shared/components/checkbox-filter/checkbox-filter';
import {DataLoaderDropdown} from '../../../shared/components/data-loader-dropdown';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {TagsInput} from '../../../shared/components/tags-input/tags-input';
import {services} from '../../../shared/services';

require('./workflow-filters.scss');

interface WorkflowFilterProps {
    workflows: models.Workflow[];
    namespace: string;
    phaseItems: string[];
    selectedPhases: string[];
    selectedLabels: string[];
    onChange: (namespace: string, selectedPhases: string[], labels: string[]) => void;
}

export class WorkflowFilters extends React.Component<WorkflowFilterProps, {}> {
    private set workflowTemplate(value: string) {
        this.setLabel(models.labels.workflowTemplate, value);
    }

    private set cronWorkflow(value: string) {
        this.setLabel(models.labels.cronWorkflow, value);
    }

    private get labelSuggestion() {
        return this.getLabelSuggestions(this.props.workflows);
    }

    public render() {
        return (
            <div className='wf-filters-container'>
                <div className='row'>
                    <div className='columns small-3 xlarge-12'>
                        <p className='wf-filters-container__title'>Namespace</p>
                        <NamespaceFilter
                            value={this.props.namespace}
                            onChange={ns => {
                                this.props.onChange(ns, this.props.selectedPhases, this.props.selectedLabels);
                            }}
                        />
                    </div>
                    <div className='columns small-3 xlarge-12'>
                        <p className='wf-filters-container__title'>Labels</p>
                        <TagsInput
                            placeholder=''
                            autocomplete={this.labelSuggestion}
                            tags={this.props.selectedLabels}
                            onChange={tags => {
                                this.props.onChange(this.props.namespace, this.props.selectedPhases, tags);
                            }}
                        />
                    </div>
                    <div className='columns small-3 xlarge-12'>
                        <p className='wf-filters-container__title'>Workflow Template</p>
                        <DataLoaderDropdown
                            load={() => services.workflowTemplate.list(this.props.namespace).then(list => list.map(x => x.metadata.name))}
                            onChange={value => (this.workflowTemplate = value)}
                        />
                    </div>
                    <div className='columns small-3 xlarge-12'>
                        <p className='wf-filters-container__title'>Cron Workflow</p>
                        <DataLoaderDropdown
                            load={() => services.cronWorkflows.list(this.props.namespace).then(list => list.map(x => x.metadata.name))}
                            onChange={value => (this.cronWorkflow = value)}
                        />
                    </div>
                    <div className='columns small-6 xlarge-12'>
                        <p className='wf-filters-container__title'>Phases</p>
                        <CheckboxFilter
                            selected={this.props.selectedPhases}
                            onChange={selected => {
                                this.props.onChange(this.props.namespace, selected, this.props.selectedLabels);
                            }}
                            items={this.getPhaseItems(this.props.workflows)}
                            type='phase'
                        />
                    </div>
                </div>
            </div>
        );
    }

    private setLabel(name: string, value: string) {
        this.props.onChange(this.props.namespace, this.props.selectedPhases, [name.concat('=' + value)]);
    }

    private getPhaseItems(workflows: models.Workflow[]) {
        const phasesMap = new Map<string, number>();
        this.props.phaseItems.forEach(value => phasesMap.set(value, 0));
        workflows.filter(wf => wf.status.phase).forEach(wf => phasesMap.set(wf.status.phase, (phasesMap.get(wf.status.phase) || 0) + 1));
        const results = new Array<{name: string; count: number}>();
        phasesMap.forEach((val, key) => {
            results.push({name: key, count: val});
        });
        return results;
    }

    private addCommonLabel(suggestions: string[]) {
        const commonLabel = new Array<string>();
        const commonLabelPool = [models.labels.cronWorkflow, models.labels.workflowTemplate, models.labels.clusterWorkflowTemplate];
        commonLabelPool.forEach(labelPrefix => {
            for (const label of suggestions) {
                if (label.startsWith(labelPrefix)) {
                    commonLabel.push(`${labelPrefix}`);
                    break;
                }
            }
        });
        return commonLabel.concat(suggestions);
    }

    private getLabelSuggestions(workflows: models.Workflow[]) {
        const suggestions = new Array<string>();
        workflows
            .filter(wf => wf.metadata.labels)
            .forEach(wf => {
                Object.keys(wf.metadata.labels).forEach(label => {
                    const value = wf.metadata.labels[label];
                    const suggestedLabel = `${label}=${value}`;
                    if (!suggestions.some(v => v === suggestedLabel)) {
                        suggestions.push(`${label}=${value}`);
                    }
                });
            });
        return this.addCommonLabel(suggestions.sort((a, b) => a.localeCompare(b)));
    }
}

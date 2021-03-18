import * as React from 'react';
import DatePicker from 'react-datepicker';
import * as models from '../../../../models';
import {CheckboxFilter} from '../../../shared/components/checkbox-filter/checkbox-filter';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {TagsInput} from '../../../shared/components/tags-input/tags-input';

import 'react-datepicker/dist/react-datepicker.css';

require('./archived-workflow-filters.scss');

interface ArchivedWorkflowFilterProps {
    workflows: models.Workflow[];
    namespace: string;
    phaseItems: string[];
    selectedPhases: string[];
    selectedLabels: string[];
    minStartedAt?: Date;
    maxStartedAt?: Date;
    onChange: (namespace: string, selectedPhases: string[], labels: string[], minStartedAt: Date, maxStartedAt: Date) => void;
}

export class ArchivedWorkflowFilters extends React.Component<ArchivedWorkflowFilterProps, {}> {
    public render() {
        return (
            <div className='wf-filters-container'>
                <div className='row'>
                    <div className='columns small-2 xlarge-12'>
                        <p className='wf-filters-container__title'>Namespace</p>
                        <NamespaceFilter
                            value={this.props.namespace}
                            onChange={ns => {
                                this.props.onChange(ns, this.props.selectedPhases, this.props.selectedLabels, this.props.minStartedAt, this.props.maxStartedAt);
                            }}
                        />
                    </div>
                    <div className='columns small-2 xlarge-12'>
                        <p className='wf-filters-container__title'>Labels</p>
                        <TagsInput
                            placeholder=''
                            autocomplete={this.getLabelSuggestions(this.props.workflows)}
                            tags={this.props.selectedLabels}
                            onChange={tags => {
                                this.props.onChange(this.props.namespace, this.props.selectedPhases, tags, this.props.minStartedAt, this.props.maxStartedAt);
                            }}
                        />
                    </div>
                    <div className='columns small-3 xlarge-12'>
                        <p className='wf-filters-container__title'>Phases</p>
                        <CheckboxFilter
                            selected={this.props.selectedPhases}
                            onChange={selected => {
                                this.props.onChange(this.props.namespace, selected, this.props.selectedLabels, this.props.minStartedAt, this.props.maxStartedAt);
                            }}
                            items={this.getPhaseItems(this.props.workflows)}
                            type='phase'
                        />
                    </div>
                    <div className='columns small-5 xlarge-12'>
                        <p className='wf-filters-container__title'>Started Time</p>
                        <DatePicker
                            selected={this.props.minStartedAt}
                            onChange={date => {
                                this.props.onChange(this.props.namespace, this.props.selectedPhases, this.props.selectedLabels, date, this.props.maxStartedAt);
                            }}
                            placeholderText='From'
                            dateFormat='dd MMM yyyy'
                            todayButton='Today'
                            className='argo-field argo-textarea'
                        />
                        <DatePicker
                            selected={this.props.maxStartedAt}
                            onChange={date => {
                                this.props.onChange(this.props.namespace, this.props.selectedPhases, this.props.selectedLabels, this.props.minStartedAt, date);
                            }}
                            placeholderText='To'
                            dateFormat='dd MMM yyyy'
                            todayButton='Today'
                            className='argo-field argo-textarea'
                        />
                    </div>
                </div>
            </div>
        );
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
        return suggestions.sort((a, b) => a.localeCompare(b));
    }
}

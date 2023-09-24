import * as React from 'react';
import {useMemo} from 'react';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';
import * as models from '../../../../models';
import {WorkflowPhase} from '../../../../models';
import {CheckboxFilter} from '../../../shared/components/checkbox-filter/checkbox-filter';
import {DataLoaderDropdown} from '../../../shared/components/data-loader-dropdown';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {TagsInput} from '../../../shared/components/tags-input/tags-input';
import {services} from '../../../shared/services';

require('./workflow-filters.scss');

interface WorkflowFilterProps {
    workflows: models.Workflow[];
    namespace: string;
    phaseItems: WorkflowPhase[];
    selectedPhases: WorkflowPhase[];
    selectedLabels: string[];
    createdAfter?: Date;
    finishedBefore?: Date;
    onChange: (namespace: string, selectedPhases: WorkflowPhase[], labels: string[], createdAfter: Date, finishedBefore: Date) => void;
}

export function WorkflowFilters(props: WorkflowFilterProps) {
    function setLabel(name: string, value: string) {
        props.onChange(props.namespace, props.selectedPhases, [name.concat('=' + value)], props.createdAfter, props.finishedBefore);
    }

    function setWorkflowTemplate(value: string) {
        setLabel(models.labels.workflowTemplate, value);
    }

    function setCronWorkflow(value: string) {
        setLabel(models.labels.cronWorkflow, value);
    }

    const labelSuggestion = useMemo(() => {
        return getLabelSuggestions(props.workflows);
    }, [props.workflows]);

    const phaseItems = useMemo(() => {
        const phasesMap = new Map<string, number>();
        props.phaseItems.forEach(value => phasesMap.set(value, 0));
        props.workflows.filter(wf => wf.status.phase).forEach(wf => phasesMap.set(wf.status.phase, (phasesMap.get(wf.status.phase) || 0) + 1));

        const results = new Array<{name: string; count: number}>();
        phasesMap.forEach((val, key) => {
            results.push({name: key, count: val});
        });
        return results;
    }, [props.workflows, props.phaseItems]);

    return (
        <div className='wf-filters-container'>
            <div className='row'>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Namespace</p>
                    <NamespaceFilter
                        value={props.namespace}
                        onChange={ns => {
                            props.onChange(ns, props.selectedPhases, props.selectedLabels, props.createdAfter, props.finishedBefore);
                        }}
                    />
                </div>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Labels</p>
                    <TagsInput
                        placeholder=''
                        autocomplete={labelSuggestion}
                        tags={props.selectedLabels}
                        onChange={tags => {
                            props.onChange(props.namespace, props.selectedPhases, tags, props.createdAfter, props.finishedBefore);
                        }}
                    />
                </div>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Workflow Template</p>
                    <DataLoaderDropdown
                        load={async () => {
                            const list = await services.workflowTemplate.list(props.namespace, []);
                            return (list.items || []).map(x => x.metadata.name);
                        }}
                        onChange={setWorkflowTemplate}
                    />
                </div>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Cron Workflow</p>
                    <DataLoaderDropdown
                        load={async () => {
                            const list = await services.cronWorkflows.list(props.namespace);
                            return list.map(x => x.metadata.name);
                        }}
                        onChange={setCronWorkflow}
                    />
                </div>
                <div className='columns small-4 xlarge-12'>
                    <p className='wf-filters-container__title'>Phases</p>
                    <CheckboxFilter
                        selected={props.selectedPhases}
                        onChange={selected => {
                            props.onChange(
                                props.namespace,
                                selected.map(x => x as WorkflowPhase),
                                props.selectedLabels,
                                props.createdAfter,
                                props.finishedBefore
                            );
                        }}
                        items={phaseItems}
                        type='phase'
                    />
                </div>
                <div className='columns small-5 xlarge-12'>
                    <p className='wf-filters-container__title'>Created Since</p>
                    <div className='wf-filters-container__content'>
                        <DatePicker
                            selected={props.createdAfter}
                            onChange={date => {
                                props.onChange(props.namespace, props.selectedPhases, props.selectedLabels, date, props.finishedBefore);
                            }}
                            placeholderText='From'
                            dateFormat='dd MMM yyyy'
                            todayButton='Today'
                            className='argo-field argo-textarea'
                        />
                        <a
                            onClick={() => {
                                props.onChange(props.namespace, props.selectedPhases, props.selectedLabels, undefined, props.finishedBefore);
                            }}>
                            <i className='fa fa-times-circle' />
                        </a>
                    </div>
                    <p className='wf-filters-container__title'>Finished Before</p>
                    <div className='wf-filters-container__content'>
                        <DatePicker
                            selected={props.finishedBefore}
                            onChange={date => {
                                props.onChange(props.namespace, props.selectedPhases, props.selectedLabels, props.createdAfter, date);
                            }}
                            placeholderText='To'
                            dateFormat='dd MMM yyyy'
                            todayButton='Today'
                            className='argo-field argo-textarea'
                        />
                        <a
                            onClick={() => {
                                props.onChange(props.namespace, props.selectedPhases, props.selectedLabels, props.createdAfter, undefined);
                            }}>
                            <i className='fa fa-times-circle' />
                        </a>
                    </div>
                </div>
            </div>
        </div>
    );
}

function addCommonLabel(suggestions: string[]) {
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

function getLabelSuggestions(workflows: models.Workflow[]) {
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
    return addCommonLabel(suggestions.sort((a, b) => a.localeCompare(b)));
}

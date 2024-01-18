import * as React from 'react';
import {useEffect, useState} from 'react';
import * as models from '../../../../models';
import {CheckboxFilter} from '../../../shared/components/checkbox-filter/checkbox-filter';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {TagsInput} from '../../../shared/components/tags-input/tags-input';

import './cron-workflow-filters.scss';

interface WorkflowFilterProps {
    cronWorkflows: models.WorkflowTemplate[];
    namespace: string;
    labels: string[];
    states: string[];
    onChange: (namespace: string, labels: string[], states: string[]) => void;
}

export function CronWorkflowFilters({cronWorkflows, namespace, labels, states, onChange}: WorkflowFilterProps) {
    const [labelSuggestion, setLabelSuggestion] = useState([]);

    useEffect(() => {
        const suggestions = new Array<string>();
        cronWorkflows
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
        setLabelSuggestion(suggestions.sort((a, b) => a.localeCompare(b)));
    }, [cronWorkflows]);

    return (
        <div className='wf-filters-container'>
            <div className='row'>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Namespace</p>
                    <NamespaceFilter
                        value={namespace}
                        onChange={ns => {
                            onChange(ns, labels, states);
                        }}
                    />
                </div>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Labels</p>
                    <TagsInput
                        placeholder=''
                        autocomplete={labelSuggestion}
                        tags={labels}
                        onChange={tags => {
                            onChange(namespace, tags, states);
                        }}
                    />
                </div>
                <div className='columns small-3 xlarge-12'>
                    <p className='wf-filters-container__title'>State</p>
                    <CheckboxFilter
                        selected={states}
                        onChange={selected => {
                            onChange(namespace, labels, selected);
                        }}
                        items={[
                            {name: 'Running', count: 0},
                            {name: 'Suspended', count: 1}
                        ]}
                        type='state'
                    />
                </div>
            </div>
        </div>
    );
}

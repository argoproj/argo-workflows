import * as React from 'react';
import {useEffect, useState} from 'react';

import * as models from '../../../../models';
import {InputFilter} from '../../../shared/components/input-filter';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {TagsInput} from '../../../shared/components/tags-input/tags-input';

import './workflow-template-filters.scss';

interface WorkflowFilterProps {
    templates: models.WorkflowTemplate[];
    namespace: string;
    namePattern: string;
    labels: string[];
    onChange: (namespace: string, namePattern: string, labels: string[]) => void;
}

export function WorkflowTemplateFilters({templates, namespace, namePattern, labels, onChange}: WorkflowFilterProps) {
    const [labelSuggestion, setLabelSuggestion] = useState([]);

    useEffect(() => {
        const suggestions = new Array<string>();
        templates
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
    }, [templates]);

    return (
        <div className='wf-filters-container'>
            <div className='row'>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Namespace</p>
                    <NamespaceFilter
                        value={namespace}
                        onChange={ns => {
                            onChange(ns, namePattern, labels);
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
                            onChange(namespace, namePattern, tags);
                        }}
                    />
                </div>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Name Pattern</p>
                    <InputFilter
                        value={namePattern}
                        name='wfnamePattern'
                        onChange={wfnamePattern => {
                            onChange(namespace, wfnamePattern, labels);
                        }}
                    />
                </div>
            </div>
        </div>
    );
}

import * as React from 'react';
import {useEffect, useState} from 'react';
import * as models from '../../../../models';
import {ClusterFilter} from '../../../shared/components/cluster-filter';
import {NamespaceFilter} from '../../../shared/components/namespace-filter';
import {TagsInput} from '../../../shared/components/tags-input/tags-input';

require('./workflow-template-filters.scss');

interface WorkflowFilterProps {
    templates: models.WorkflowTemplate[];
    cluster: string;
    namespace: string;
    labels: string[];
    onChange: (cluster: string, namespace: string, labels: string[]) => void;
}

export const WorkflowTemplateFilters = ({templates, cluster, namespace, labels, onChange}: WorkflowFilterProps) => {
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
                    <p className='wf-filters-container__title'>Cluster</p>
                    <ClusterFilter
                        value={cluster}
                        onChange={v => {
                            onChange(v, namespace, labels);
                        }}
                    />
                </div>
                <div className='columns small-2 xlarge-12'>
                    <p className='wf-filters-container__title'>Namespace</p>
                    <NamespaceFilter
                        value={namespace}
                        onChange={ns => {
                            onChange(cluster, ns, labels);
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
                            onChange(cluster, namespace, tags);
                        }}
                    />
                </div>
            </div>
        </div>
    );
};

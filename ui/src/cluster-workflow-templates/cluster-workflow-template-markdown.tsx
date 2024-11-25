import * as React from 'react';

import {ANNOTATION_DESCRIPTION, ANNOTATION_TITLE} from '../shared/annotations';
import {SuspenseReactMarkdownGfm} from '../shared/components/suspense-react-markdown-gfm';
import {ClusterWorkflowTemplate} from '../shared/models';

require('./cluster-workflow-template-markdown.scss');

interface ClusterWorkflowTemplateMarkdownProps {
    workflow: ClusterWorkflowTemplate;
}

export function ClusterWorkflowTemplateMarkdown(props: ClusterWorkflowTemplateMarkdownProps) {
    const wf = props.workflow;
    // title + description vars
    const title = wf.metadata.annotations?.[ANNOTATION_TITLE] ?? wf.metadata.name;
    const description = (wf.metadata.annotations?.[ANNOTATION_DESCRIPTION] && `\n${wf.metadata.annotations[ANNOTATION_DESCRIPTION]}`) || '';
    const hasAnnotation = title !== wf.metadata.name || description !== '';
    const markdown = `${title}${description}`;

    return <div className='wf-rows-name'>{hasAnnotation ? <SuspenseReactMarkdownGfm markdown={markdown} /> : markdown}</div>;
}

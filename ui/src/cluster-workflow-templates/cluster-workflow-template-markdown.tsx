import * as React from 'react';

import {ANNOTATION_DESCRIPTION, ANNOTATION_TITLE} from '../shared/annotations';
import {SuspenseReactMarkdownGfm} from '../shared/components/suspense-react-markdown-gfm';
import {ClusterWorkflowTemplate} from '../shared/models';
import {escapeInvalidMarkdown} from '../workflows/utils';

require('./cluster-workflow-template-markdown.scss');

interface ClusterWorkflowTemplateMarkdownProps {
    workflow: ClusterWorkflowTemplate;
}

export function ClusterWorkflowTemplateMarkdown(props: ClusterWorkflowTemplateMarkdownProps) {
    const wf = props.workflow;
    // title + description vars
    const title = (wf.metadata.annotations?.[ANNOTATION_TITLE] && `${escapeInvalidMarkdown(wf.metadata.annotations[ANNOTATION_TITLE])}`) ?? wf.metadata.name;
    const description = (wf.metadata.annotations?.[ANNOTATION_DESCRIPTION] && `\n${escapeInvalidMarkdown(wf.metadata.annotations[ANNOTATION_DESCRIPTION])}`) || '';
    const markdown = `${title}${description}`;

    return (
        <div className={description.length ? 'wf-rows-name' : ''} aria-valuetext={markdown}>
            <SuspenseReactMarkdownGfm markdown={markdown} />
        </div>
    );
}

import * as React from 'react';
import {Link} from 'react-router-dom';

import {ANNOTATION_DESCRIPTION, ANNOTATION_TITLE} from '../shared/annotations';
import {uiUrl} from '../shared/base';
import {SuspenseReactMarkdownGfm} from '../shared/components/suspense-react-markdown-gfm';
import {Timestamp} from '../shared/components/timestamp';
import {WorkflowTemplate} from '../shared/models';
import {escapeInvalidMarkdown} from '../workflows/utils';

require('./workflow-template-row.scss');

interface WorkflowTemplateRowProps {
    workflow: WorkflowTemplate;
    displayISOFormat: boolean;
}

export function WorkflowTemplateRow(props: WorkflowTemplateRowProps) {
    const wf = props.workflow;
    // title + description vars
    const title = (wf.metadata.annotations?.[ANNOTATION_TITLE] && `${escapeInvalidMarkdown(wf.metadata.annotations[ANNOTATION_TITLE])}`) ?? wf.metadata.name;
    const description = (wf.metadata.annotations?.[ANNOTATION_DESCRIPTION] && `\n${escapeInvalidMarkdown(wf.metadata.annotations[ANNOTATION_DESCRIPTION])}`) || '';
    const hasAnnotation = title !== wf.metadata.name && description !== '';
    const markdown = `${title}${description}`;

    return (
        <div className='workflow-templates-list__row-container'>
            <div className='row argo-table-list__row'>
                <div className='columns small-1'>
                    <i className='fa fa-clone' />
                </div>
                <Link to={{pathname: uiUrl(`workflow-templates/${wf.metadata.namespace}/${wf.metadata.name}`)}} className='columns small-5'>
                    {hasAnnotation || description.length ? (
                        <div className='wf-rows-name'>
                            <SuspenseReactMarkdownGfm markdown={markdown} />
                        </div>
                    ) : (
                        <span>
                            <SuspenseReactMarkdownGfm markdown={markdown} />
                        </span>
                    )}
                </Link>
                <div className='columns small-3'>{wf.metadata.namespace}</div>
                <div className='columns small-3'>
                    <Timestamp date={wf.metadata.creationTimestamp} displayISOFormat={props.displayISOFormat} />
                </div>
            </div>
        </div>
    );
}

import * as kubernetes from 'argo-ui/src/models/kubernetes';
import * as React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import {ANNOTATION_DESCRIPTION, ANNOTATION_TITLE} from '../../../shared/annotations';

require('./workflows-row.scss');

export const WorkflowsRowName = ({metadata}: {metadata: kubernetes.ObjectMeta}) => {
    const title = (metadata.annotations && metadata.annotations[ANNOTATION_TITLE]) || metadata.name;
    const description = (metadata.annotations && metadata.annotations[ANNOTATION_DESCRIPTION] && `\n${metadata.annotations[ANNOTATION_DESCRIPTION]}`) || '';
    const markdown = `${title}${description}`;
    return (
        <div className='wf-rows-name'>
            <ReactMarkdown components={{p: React.Fragment}} remarkPlugins={[remarkGfm]}>
                {markdown}
            </ReactMarkdown>
        </div>
    );
};

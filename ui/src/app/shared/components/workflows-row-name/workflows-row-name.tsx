import * as kubernetes from 'argo-ui/src/models/kubernetes';
import * as React from 'react';
import {ANNOTATION_DESCRIPTION, ANNOTATION_TITLE} from '../../../shared/annotations';

require('./workflows-row-name.scss');

export const WorkflowsRowName = ({metadata}: {metadata: kubernetes.ObjectMeta}) => {
    const title = (metadata.annotations && metadata.annotations[ANNOTATION_TITLE]) || metadata.name;
    const description = (metadata.annotations && metadata.annotations[ANNOTATION_DESCRIPTION] && `\n${metadata.annotations[ANNOTATION_DESCRIPTION]}`) || '';
    const content = `${title}${description}`;
    return <div className='wf-rows-name'>{content}</div>;
};

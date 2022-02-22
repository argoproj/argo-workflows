import * as React from 'react';
import {uiUrl} from '../../shared/base';
import {LinkButton} from '../../shared/components/link-button';

export const CronWorkflowLink = (props: {namespace: string; name: string}) => (
    <LinkButton to={uiUrl('cron-workflows/' + props.namespace + '/' + props.name)}>
        <i className='fa fa-clock' /> {props.name}
    </LinkButton>
);

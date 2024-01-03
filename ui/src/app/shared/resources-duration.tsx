import * as React from 'react';
import {denominator, formatDuration} from './duration';

interface Props {
    resourcesDuration: {[resource: string]: number};
}

export function ResourcesDuration(props: Props) {
    return (
        <>
            {props.resourcesDuration &&
                Object.entries(props.resourcesDuration)
                    .map(([resource, duration]) => formatDuration(duration, 1) + '*(' + denominator(resource) + ' ' + resource + ')')
                    .join(',')}{' '}
            <a href='https://argo-workflows.readthedocs.io/en/latest/resource-duration/'>
                <i className='fa fa-info-circle' />
            </a>
        </>
    );
}

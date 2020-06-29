import * as React from 'react';
import {denominator, formatDuration} from './duration';

interface Props {
    resourcesDuration: {[resource: string]: number};
}

export class ResourcesDuration extends React.Component<Props> {
    public render() {
        return (
            <>
                {this.props.resourcesDuration &&
                    Object.entries(this.props.resourcesDuration)
                        .map(([resource, duration]) => formatDuration(duration, 1) + '*(' + denominator(resource) + ' ' + resource + ')')
                        .join(',')}{' '}
                <a href='https://github.com/argoproj/argo/blob/master/docs/resource-duration.md'>
                    <i className='fa fa-info-circle' />
                </a>
            </>
        );
    }
}

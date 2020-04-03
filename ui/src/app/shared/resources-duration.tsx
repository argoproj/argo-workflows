import * as React from 'react';
import {formatDuration} from './duration';

interface Props {
    resourcesDuration: {[resource: string]: number};
}

export class ResourcesDuration extends React.Component<Props> {
    public render() {
        function denominator(resource: string) {
            switch (resource) {
                case 'memory':
                    return '1Gi';
                case 'storage':
                    return '10Gi';
                case 'ephemeral-storage':
                    return '10Gi';
                default:
                    return '1';
            }
        }

        return (
            <>
                {this.props.resourcesDuration &&
                    Object.entries(this.props.resourcesDuration)
                        .map(([resource, duration]) => formatDuration(duration) + '*(' + denominator(resource) + ' ' + resource + ')')
                        .join(',')}{' '}
                <a href='https://github.com/argoproj/argo/blob/master/docs/resource-duration.md'>
                    <i className='fa fa-info-circle' />
                </a>
            </>
        );
    }
}

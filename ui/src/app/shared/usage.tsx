import * as React from 'react';
import {denominator, formatDuration} from './duration';

interface Props {
    usage: {[resource: string]: number};
}

export class Usage extends React.Component<Props> {
    public render() {
        return (
            <>
                {this.props.usage &&
                    Object.entries(this.props.usage)
                        .map(([resource, duration]) => formatDuration(duration) + '*(' + denominator(resource) + ' ' + resource + ')')
                        .join(',')}{' '}
                <a href='https://github.com/argoproj/argo/blob/master/docs/usage.md'>
                    <i className='fa fa-info-circle' />
                </a>
            </>
        );
    }
}

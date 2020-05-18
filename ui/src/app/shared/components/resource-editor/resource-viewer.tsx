import * as React from 'react';
import {stringify} from './resource';

require('./resource.scss');

interface Props<T> {
    value: T;
    type?: string;
}

export class ResourceViewer<T> extends React.Component<Props<T>> {
    public render() {
        return <div className='resource'>{stringify(this.props.value, this.props.type || 'yaml')}</div>;
    }
}

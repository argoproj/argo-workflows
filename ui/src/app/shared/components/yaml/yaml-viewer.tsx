import * as React from 'react';

require('./yaml.scss');

interface Props {
    value: string;
}

export class YamlViewer extends React.Component<Props> {
    public render() {
        return <div className='yaml'>{this.props.value}</div>;
    }
}

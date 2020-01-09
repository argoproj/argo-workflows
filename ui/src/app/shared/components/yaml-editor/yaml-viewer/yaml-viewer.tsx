import * as React from 'react';

interface Props {
    yaml: string;
}

export class YamlViewer extends React.Component<Props> {
    public render() {
        return (
            <div style={{margin: 10}}>
                <pre>{this.props.yaml}</pre>
            </div>
        );
    }
}

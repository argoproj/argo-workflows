import * as React from 'react';

interface Props {
    title: string;
}

// https://designsystem.quickbooks.com/pattern/zero-states/
export class ZeroState extends React.Component<Props> {
    public render() {
        return (
            <div style={{margin: '50px'}}>
                <div className='white-box'>
                    <h4>{this.props.title}</h4>
                    {this.props.children}
                </div>
            </div>
        );
    }
}

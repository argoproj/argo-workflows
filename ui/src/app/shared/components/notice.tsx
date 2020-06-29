import * as React from 'react';

export class Notice extends React.Component {
    public render() {
        return (
            <div style={{marginTop: 20, marginBottom: 20}}>
                <div className='white-box' style={{padding: 20}}>
                    {this.props.children}
                </div>
            </div>
        );
    }
}

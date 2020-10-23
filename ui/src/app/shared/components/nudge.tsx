import * as React from 'react';
import {Notice} from './notice';

export class Nudge extends React.Component<{key: string}, {closed: boolean}> {
    constructor(props: Readonly<{key: string}>) {
        super(props);
        this.state = {closed: localStorage.getItem(props.key) !== null};
    }

    public render() {
        return (
            !this.state.closed && (
                <Notice>
                    {this.props.children}
                    <span className='fa-pull-right'>
                        <a onClick={() => this.close()}>
                            <i className='fa fa-times' />
                        </a>{' '}
                    </span>
                </Notice>
            )
        );
    }

    private close() {
        this.setState({closed: true});
        localStorage.setItem(this.props.key, '{}');
    }
}

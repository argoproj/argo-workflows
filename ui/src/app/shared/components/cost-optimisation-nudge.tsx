import * as React from 'react';
import {Notice} from './notice';

interface Props {
    name: string;
}
interface State {
    closed: boolean;
}

export class CostOptimisationNudge extends React.Component<Props, State> {
    constructor(props: Readonly<Props>) {
        super(props);
        this.state = {closed: localStorage.getItem(this.key) !== null};
    }

    public render() {
        return (
            !this.state.closed && (
                <Notice>
                    <i className='fa fa-money-bill-alt status-icon--pending' /> {this.props.children} <a href='https://argoproj.github.io/argo/cost-optimisation/'>Learn more</a>
                    <span className='fa-pull-right'>
                        <a onClick={() => this.close()}>
                            <i className='fa fa-times' />
                        </a>{' '}
                    </span>
                </Notice>
            )
        );
    }

    private get key() {
        return 'cost-optimization-nude/' + this.props.name;
    }

    private close() {
        this.setState({closed: true});
        localStorage.setItem(this.key, '{}');
    }
}

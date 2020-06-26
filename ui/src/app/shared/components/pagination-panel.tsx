import * as React from 'react';
import {Pagination, parseLimit} from '../pagination';

export class PaginationPanel extends React.Component<{pagination: Pagination; onChange: (pagination: Pagination) => void}> {
    public render() {
        return (
            <p>
                <button
                    disabled={!this.props.pagination.offset}
                    className='argo-button argo-button--base-o'
                    onClick={() => this.props.onChange({limit: this.props.pagination.limit})}>
                    First page
                </button>
                <button
                    disabled={!this.props.pagination.nextOffset}
                    className='argo-button argo-button--base-o'
                    onClick={() =>
                        this.props.onChange({
                            limit: this.props.pagination.limit,
                            offset: this.props.pagination.nextOffset
                        })
                    }>
                    Next page <i className='fa fa-chevron-right' />
                </button>
                {this.props.pagination.limit ? (
                    <>
                        <span className={'fa fa-exclamation-triangle'} style={{color: '#d7b700'}} />
                        Workflows cannot be globally sorted when paginated
                    </>
                ) : (
                    <span />
                )}
                <small className='fa-pull-right'>
                    <select
                        className='small'
                        onChange={e =>
                            this.props.onChange({
                                offset: this.props.pagination.offset,
                                limit: parseLimit(e.target.value)
                            })
                        }
                        value={this.props.pagination.limit || 'all'}>
                        {[5, 10, 20, 50, 100, 500, 'all'].map(limit => (
                            <option key={limit} value={limit}>
                                {limit}
                            </option>
                        ))}
                    </select>{' '}
                    results per page
                </small>
            </p>
        );
    }
}

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
                        onChange={e => {
                            const limit = parseLimit(e.target.value);
                            const newValue: Pagination = {limit};

                            // Only return the offset if we're actually going to be limiting
                            // the results we're requesting.  If we're requesting all records,
                            // we should not skip any by setting an offset.
                            if (limit) {
                                newValue.offset = this.props.pagination.offset;
                            }

                            this.props.onChange(newValue);
                        }}
                        value={this.props.pagination.limit || 0}>
                        {[5, 10, 20, 50, 100, 500, 0].map(limit => (
                            <option key={limit} value={limit}>
                                {limit === 0 ? 'all' : limit}
                            </option>
                        ))}
                    </select>{' '}
                    results per page
                </small>
            </p>
        );
    }
}

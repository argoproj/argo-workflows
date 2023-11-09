import * as React from 'react';
import {Pagination, parseLimit} from '../pagination';
import {WarningIcon} from './fa-icons';

export function PaginationPanel(props: {pagination: Pagination; onChange: (pagination: Pagination) => void; numRecords: number}) {
    return (
        <p style={{paddingBottom: '45px'}}>
            <button disabled={!props.pagination.offset} className='argo-button argo-button--base-o' onClick={() => props.onChange({limit: props.pagination.limit})}>
                First page
            </button>
            <button
                disabled={!props.pagination.nextOffset}
                className='argo-button argo-button--base-o'
                onClick={() =>
                    props.onChange({
                        limit: props.pagination.limit,
                        offset: props.pagination.nextOffset
                    })
                }>
                Next page <i className='fa fa-chevron-right' />
            </button>
            {/* if pagination is used, and we're either not on the first page, or are on the first page and have more records than the page limit */}
            {props.pagination.limit > 0 && (props.pagination.offset || (!props.pagination.offset && props.numRecords >= props.pagination.limit)) ? (
                <>
                    <WarningIcon /> Workflows cannot be globally sorted when paginated
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
                        // The offset must be initialized whenever the pagination limit is changed.
                        if (limit) {
                            newValue.offset = '';
                        }

                        props.onChange(newValue);
                    }}
                    value={props.pagination.limit || 0}>
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

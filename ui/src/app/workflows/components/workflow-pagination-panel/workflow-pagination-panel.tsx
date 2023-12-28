import * as React from 'react';
import {WorkflowsPagination} from '../../../../models';
import {WarningIcon} from '../../../shared/components/fa-icons';
import {parseLimit} from '../../../shared/pagination';

export function WorkflowPaginationPanel(props: {pagination: WorkflowsPagination; onChange: (pagination: WorkflowsPagination) => void; numRecords: number}) {
    const isFirstDisabled = !props.pagination.wfOffset && !props.pagination.archivedOffset;
    const isNextDisabled = !props.pagination.nextWfOffset && !props.pagination.nextArchivedOffset;
    return (
        <p style={{paddingBottom: '45px'}}>
            <button
                disabled={isFirstDisabled}
                className='argo-button argo-button--base-o'
                onClick={() => props.onChange({limit: props.pagination.limit, wfOffset: '', archivedOffset: '', nextWfOffset: '', nextArchivedOffset: ''})}>
                First page
            </button>
            <button
                disabled={isNextDisabled}
                className='argo-button argo-button--base-o'
                onClick={() =>
                    props.onChange({
                        limit: props.pagination.limit,
                        wfOffset: props.pagination.nextWfOffset,
                        archivedOffset: props.pagination.nextArchivedOffset,
                        nextWfOffset: '',
                        nextArchivedOffset: ''
                    })
                }>
                Next page <i className='fa fa-chevron-right' />
            </button>
            {/* if pagination is used, and we're either not on the first page, or are on the first page and have more records than the page limit */}
            {props.pagination.limit > 0 && props.numRecords >= props.pagination.limit ? (
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
                        const newValue: WorkflowsPagination = {limit, wfOffset: '', archivedOffset: '', nextWfOffset: '', nextArchivedOffset: ''};
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

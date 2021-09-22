import {Ticker} from 'argo-ui';
import * as React from 'react';
import {ago} from '../duration';

export const Timestamp = ({date}: {date: Date | string | number}) => {
    return (
        <span>
            {date === null || date === undefined ? (
                '-'
            ) : (
                <span title={date.toString()}>
                    <Ticker intervalMs={1000}>{() => ago(new Date(date))}</Ticker>
                </span>
            )}
        </span>
    );
};

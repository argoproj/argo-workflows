import {Ticker} from 'argo-ui/src/components/ticker';
import * as React from 'react';

import {ago} from '../duration';

export function Timestamp({date}: {date: Date | string | number}) {
    const tooltip = (utc: Date | string | number) => {
        return utc.toString() + '\n' + new Date(utc.toString()).toLocaleString();
    };
    return (
        <span>
            {date === null || date === undefined ? (
                '-'
            ) : (
                <span title={tooltip(date)}>
                    <Ticker intervalMs={1000}>{() => ago(new Date(date))}</Ticker>
                </span>
            )}
        </span>
    );
}

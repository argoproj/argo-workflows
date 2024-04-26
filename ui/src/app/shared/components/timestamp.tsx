import {Ticker} from 'argo-ui';
import * as React from 'react';

import {ago} from '../duration';

export function Timestamp({date, displayFullDate = false}: {date: Date | string | number; displayFullDate?: boolean}) {
    const tooltip = (utc: Date | string | number) => {
        return utc.toString() + '\n' + new Date(utc.toString()).toLocaleString();
    };
    return (
        <span>
            {date === null || date === undefined ? (
                '-'
            ) : (
                <span title={tooltip(date)}>
                    <Ticker intervalMs={1000}>{() => (displayFullDate ? new Date(date).toISOString() : ago(new Date(date)))}</Ticker>
                </span>
            )}
        </span>
    );
}

import {Ticker} from 'argo-ui/src/components/ticker';
import {Tooltip} from 'argo-ui/src/components/tooltip/tooltip';
import * as React from 'react';

import {ago} from '../duration';
import useTimestamp, {TIMESTAMP_KEYS} from '../use-timestamp';

export function Timestamp({date, timestampKey, displayLocalDateTime}: {date: Date | string | number; timestampKey: TIMESTAMP_KEYS; displayLocalDateTime?: boolean}) {
    const {displayISOFormat, setDisplayISOFormat} = useTimestamp(timestampKey);

    if (date === null || date === undefined) return <span>-</span>;

    return (
        <span>
            <span title={`${date.toString()} (${ago(new Date(date))})`}>
                {displayISOFormat ? (
                    new Date(date.toString()).toISOString()
                ) : (
                    <>
                        {displayLocalDateTime ? (
                            <>
                                {new Date(date.toString()).toLocaleString()} (<Ticker intervalMs={1000}>{() => ago(new Date(date))}</Ticker>)
                            </>
                        ) : (
                            <Ticker intervalMs={1000}>{() => ago(new Date(date))}</Ticker>
                        )}
                    </>
                )}
            </span>
            <Tooltip content='Switch time format'>
                <a>
                    <i
                        className={'fa fa-clock'}
                        style={{marginLeft: 4}}
                        onClick={e => {
                            e.stopPropagation();
                            e.preventDefault();
                            setDisplayISOFormat(!displayISOFormat);
                        }}
                    />
                </a>
            </Tooltip>
        </span>
    );
}

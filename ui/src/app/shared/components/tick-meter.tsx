import * as React from 'react';
import {useEffect, useState} from 'react';

export const TickMeter = ({value}: {value: number}) => {
    const [change, setChange] = useState<number>(0);
    const [previousValue, setPreviousValue] = useState<number>();
    useEffect(() => {
        if (previousValue && value) {
            setChange(value - previousValue);
        }
        setPreviousValue(value);
    }, [value]);
    return (
        <>
            {change === 0 ? ' ' : change > 0 ? <i className='fas fa-caret-up' /> : <i className='fas fa-caret-down' />}
            {value}
        </>
    );
};

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
            {value && value.toLocaleString()}
            {change !== 0 && (
                <small style={{color: 'gray'}}>
                    ({change > 0 ? '+' : ''}
                    {change.toLocaleString()})
                </small>
            )}
        </>
    );
};

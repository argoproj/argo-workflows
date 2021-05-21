import * as React from 'react';
import {useEffect, useState} from 'react';
import Sparkline from 'react-sparkline-svg';

export const SparkMeter = ({value}: {value: number}) => {
    const [values, setValues] = useState<number[]>([]);
    useEffect(() => {
        if (!isNaN(value)) {
            setValues(values.concat([value]));
        }
    }, [value]);

    const min = Math.min(...values);

    return values.length > 1 && <Sparkline values={values.map(v => v - min)} height='20px' width='100px' fill='rgba(128, 128, 128, 0.05)' />;
};

import * as React from 'react';

export const ProgressLine = (props: {progress: number; width: number; height: number}) => (
    <svg width={props.width} height={props.height}>
        <rect width={props.width} height={props.height} rx={4} fill='gray' />
        <rect x={2} y={2} width={(props.width - 4) * Math.min(1, props.progress)} height={props.height - 4} fill='white' rx={2} />
    </svg>
);

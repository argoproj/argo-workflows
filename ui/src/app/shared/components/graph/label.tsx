import * as React from 'react';

export const formatLabel = (label: string) => {
    if (!label) {
        return;
    }
    const maxPerLine = 14;
    if (label.length <= maxPerLine) {
        return <tspan>{label}</tspan>;
    }
    if (label.length <= maxPerLine * 2) {
        return (
            <>
                <tspan x={0} dy='-0.2em'>
                    {label.substr(0, label.length / 2)}
                </tspan>
                <tspan x={0} dy='1.2em'>
                    {label.substr(label.length / 2)}
                </tspan>
            </>
        );
    }
    return (
        <>
            <tspan x={0} dy='-0.2em'>
                {label.substr(0, maxPerLine - 2)}..
            </tspan>
            <tspan x={0} dy='1.2em'>
                {label.substr(label.length + 1 - maxPerLine)}
            </tspan>
        </>
    );
};

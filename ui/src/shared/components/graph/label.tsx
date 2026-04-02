import * as React from 'react';

export function formatLabel(label: string) {
    if (!label) {
        return;
    }
    const maxPerLine = 14;
    if (label.length <= maxPerLine) {
        return <tspan>{label}</tspan>;
    }
    if (label.length <= maxPerLine * 2) {
        const split = label.split('-');
        if (split.length >= 2) {
            let bestDiff = Number.MAX_VALUE;
            let bestI = -1;
            for (let i = 1; i <= split.length; i++) {
                const firstHalf = split.slice(0, i).join('-');
                const secondHalf = split.slice(i).join('-');
                const diff = Math.abs(firstHalf.length - secondHalf.length);
                if (diff < bestDiff) {
                    bestDiff = diff;
                    bestI = i;
                }
            }
            return (
                <>
                    <tspan x={0} dy='-0.2em'>
                        {split.slice(0, bestI).join('-') + '-'}
                    </tspan>
                    <tspan x={0} dy='1.2em'>
                        {split.slice(bestI).join('-')}
                    </tspan>
                </>
            );
        }
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
}

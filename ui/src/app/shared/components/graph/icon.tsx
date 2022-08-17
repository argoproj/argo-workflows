import * as React from 'react';
import {Icon} from '../icon';
import {icons} from '../icons';

export const GraphIcon = ({nodeSize, progress, icon}: {icon: Icon; progress?: number; nodeSize: number}) => {
    if (!progress) {
        return (
            <text className='fa icon' style={{fontSize: nodeSize / 2}}>
                {icons[icon]}
            </text>
        );
    }
    const radius = nodeSize / 4;
    const offset = (2 * Math.PI * 3) / 4;
    const theta0 = offset;
    // clip the line to min 5% max 95% so something always renders
    const theta1 = 2 * Math.PI * Math.max(0.05, Math.min(0.95, progress)) + offset;
    const start = {x: radius * Math.cos(theta0), y: radius * Math.sin(theta0)};
    const end = {x: radius * Math.cos(theta1), y: radius * Math.sin(theta1)};
    const theta = theta1 - theta0;
    const largeArcFlag = theta > Math.PI ? 1 : 0;
    const sweepFlag = 1;
    return <path className='icon' d={`M${start.x},${start.y} A${radius},${radius} 0 ${largeArcFlag} ${sweepFlag} ${end.x},${end.y}`} />;
};

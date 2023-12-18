import * as React from 'react';
import {ReactChild} from 'react';

import './inline-table.scss';

interface TableProps {
    rows: Row[];
}

interface Row {
    left: ReactChild;
    right: ReactChild;
}

export function InlineTable(props: TableProps) {
    return (
        <div className='it'>
            {props.rows.map((row, i) => {
                return (
                    <div key={i} className='it--row'>
                        <div className='it--col it--key'>{row.left}</div>
                        <div className='it--col'>{row.right}</div>
                    </div>
                );
            })}
        </div>
    );
}

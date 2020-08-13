import * as React from 'react';
import {ReactChild} from 'react';
require('./inline-table.scss');

interface TableProps {
    rows: Row[];
}

interface Row {
    left: ReactChild;
    right: ReactChild;
}

export class InlineTable extends React.Component<TableProps> {
    public render() {
        return this.props.rows.map((row, i) => {
            return (
                <div key={i}>
                    <div>{row.left}</div>
                    <div>{row.right}</div>
                </div>
            );
        });
    }
}

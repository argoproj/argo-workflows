import * as React from 'react';
import Moment from 'react-moment';

export const Timestamp = ({date}: {date: string | number}) => {
    return (
        <span>
            {date === null ? (
                '-'
            ) : (
                <Moment fromNow={true} withTitle={true}>
                    {date}
                </Moment>
            )}
        </span>
    );
};

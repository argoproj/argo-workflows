import * as React from 'react';
import Moment from 'react-moment';

export const Timestamp = ({date}: {date: string | number}) => {
    return (
        <span>
            {date === null ? (
                '-'
            ) : (
                <React.Fragment>
                    <Moment fromNow={true}>{date}</Moment> (<Moment local={true}>{date}</Moment>)
                </React.Fragment>
            )}
        </span>
    );
};

import * as React from 'react';

import {Consumer} from '../context';

/**
 * @deprecated Use React hooks instead.
 */
export const Query = (props: {children: (params: URLSearchParams) => React.ReactNode}) => (
    <Consumer>{ctx => props.children(new URLSearchParams(ctx.history.location.search))}</Consumer>
);

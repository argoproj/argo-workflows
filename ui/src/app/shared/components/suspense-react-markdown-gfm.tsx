import React from 'react';

import {Loading} from './loading';

// lazy load ReactMarkdown (and remark-gfm) as it is a large optional component (which can be split into a separate bundle)
const LazyReactMarkdownGfm = React.lazy(() => {
    return import(/* webpackChunkName: "react-markdown-plus-gfm" */ './_react-markdown-gfm');
});

export function SuspenseReactMarkdownGfm(props: {markdown: string}) {
    return (
        <React.Suspense fallback={<Loading />}>
            <LazyReactMarkdownGfm markdown={props.markdown} />
        </React.Suspense>
    );
}

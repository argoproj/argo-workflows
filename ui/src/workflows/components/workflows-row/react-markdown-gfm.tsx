import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';

import {openLinkWithKey} from '../../../shared/components/links';

export function ReactMarkdownGfm({markdown}: {markdown: string}) {
    return (
        <ReactMarkdown components={{p: React.Fragment, a: NestedAnchor}} remarkPlugins={[remarkGfm]}>
            {markdown}
        </ReactMarkdown>
    );
}
export default ReactMarkdownGfm; // for lazy loading

function NestedAnchor(props: React.ComponentProps<'a'>) {
    return (
        <a
            {...props}
            onClick={ev => {
                ev.preventDefault(); // don't bubble up
                openLinkWithKey(props.href); // eslint-disable-line react/prop-types -- it's not interpreting the prop types correctly
            }}
        />
    );
}

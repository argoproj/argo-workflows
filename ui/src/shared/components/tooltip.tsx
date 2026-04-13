import {Tooltip as ArgoTooltip} from 'argo-ui/src/components/tooltip/tooltip';
import React from 'react';
import ReactMarkdown from 'react-markdown';
import remarkBreaks from 'remark-breaks';
import remarkGfm from 'remark-gfm';

import {openLinkWithKey} from './links';

type TooltipProps = React.ComponentProps<typeof ArgoTooltip>;

function NestedAnchor(props: React.ComponentProps<'a'>) {
    return (
        <a
            {...props}
            onClick={ev => {
                ev.preventDefault();
                openLinkWithKey(props.href);
            }}
        />
    );
}

export function Tooltip({content, ...props}: TooltipProps) {
    const renderedContent =
        typeof content === 'string' ? (
            <ReactMarkdown components={{a: NestedAnchor}} remarkPlugins={[remarkGfm, remarkBreaks]}>
                {content}
            </ReactMarkdown>
        ) : (
            content
        );
    return <ArgoTooltip content={renderedContent} {...props} />;
}

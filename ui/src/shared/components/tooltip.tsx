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
                // eslint-disable-next-line react/prop-types
                if (props.href) {
                    openLinkWithKey(props.href); // eslint-disable-line react/prop-types
                }
            }}
        />
    );
}

// eslint-disable-next-line react/prop-types
export function Tooltip({content, ...props}: TooltipProps) {
    const isMarkdown = typeof content === 'string';
    const renderedContent = isMarkdown ? (
        <ReactMarkdown components={{a: NestedAnchor}} remarkPlugins={[remarkGfm, remarkBreaks]}>
            {content as string}
        </ReactMarkdown>
    ) : (
        content
    );
    return <ArgoTooltip content={renderedContent} maxWidth={isMarkdown ? '50vw' : undefined} {...props} />;
}

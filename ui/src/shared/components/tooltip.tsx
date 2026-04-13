import {Tooltip as ArgoTooltip} from 'argo-ui/src/components/tooltip/tooltip';
import React from 'react';

// Use the synchronous variant intentionally: tooltips must render immediately on hover.
// The lazy-loaded suspense variant would show a loading indicator inside the tooltip,
// which degrades UX.
import {ReactMarkdownGfm} from './_react-markdown-gfm';

type TooltipProps = React.ComponentProps<typeof ArgoTooltip>;

export function Tooltip({content, ...props}: TooltipProps) {
    const renderedContent = typeof content === 'string' ? <ReactMarkdownGfm markdown={content} /> : content;
    return <ArgoTooltip content={renderedContent} {...props} />;
}

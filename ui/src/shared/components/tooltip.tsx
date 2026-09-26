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
    // `boundary` is the tippy v5 (Popper 1) top-level API; on a v6 bump this
    // moves into `popperOptions` and 'viewport' is no longer a valid value.
    return <ArgoTooltip content={renderedContent} maxWidth={isMarkdown ? '50vw' : undefined} boundary='viewport' {...props} />;
}

// `TooltipIcon` is the shared, accessible trigger for icon-only description
// tooltips (e.g. the question-circle "more info" icon next to a label).
//
// It renders a native <button> instead of a bare <i>/<span> so that:
//   - it is reachable via keyboard (Tab) without an explicit tabIndex hack,
//   - it has an accessible name via `aria-label`, defaulting to the tooltip
//     content itself when that content is a plain string,
//   - the underlying Tippy-based Tooltip already opens on `focus` as well as
//     `mouseenter` (its default trigger), so keyboard users get the same
//     content sighted mouse users do.
//
// The `fa`/icon classes are applied directly to the button, and default
// browser button chrome (border/background/padding) is reset inline so the
// visual presentation matches the old icon-only trigger.
export interface TooltipIconProps {
    /** Tooltip content. Also used to derive the default aria-label when it's a plain string. */
    content: TooltipProps['content'];
    /** Font Awesome icon class, defaults to the standard "more info" glyph. */
    icon?: string;
    /** Explicit accessible name. Required when `content` is not a plain string. */
    label?: string;
    style?: React.CSSProperties;
    className?: string;
    /** Additional props forwarded to the underlying Tooltip (e.g. `arrow`, `placement`). */
    tooltipProps?: Omit<TooltipProps, 'content' | 'children'>;
}

const tooltipIconResetStyle: React.CSSProperties = {
    background: 'none',
    border: 0,
    padding: 0,
    margin: 0,
    font: 'inherit',
    color: 'inherit',
    cursor: 'pointer',
    lineHeight: 'inherit',
    verticalAlign: 'baseline'
};

export function TooltipIcon({content, icon = 'fa-question-circle', label, style, className, tooltipProps}: TooltipIconProps) {
    const accessibleLabel = label || (typeof content === 'string' && content.length > 0 ? content : 'More information');
    return (
        <Tooltip content={content} {...tooltipProps}>
            <button type='button' className={['fa', icon, className].filter(Boolean).join(' ')} aria-label={accessibleLabel} style={{...tooltipIconResetStyle, ...style}} />
        </Tooltip>
    );
}

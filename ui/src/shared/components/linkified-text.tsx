import LinkifyIt from 'linkify-it';
import React from 'react';

interface Props {
    text: string;
}

const linkify = new LinkifyIt();

export default function LinkifiedText({text}: Props) {
    const matches = linkify.match(text);

    if (!matches) {
        return <>{text}</>;
    }

    const parts = [];
    let lastIndex = 0;

    matches.forEach(match => {
        if (match.index > lastIndex) {
            parts.push(<span key={`text-${lastIndex}`}>{text.slice(lastIndex, match.index)}</span>);
        }
        parts.push(
            <a key={`link-${match.index}-${match.text}`} href={match.url} target='_blank' rel='noopener noreferrer' className='underline'>
                {match.text}
            </a>
        );
        lastIndex = match.lastIndex;
    });

    if (lastIndex < text.length) {
        parts.push(<span key={'text-end'}>{text.slice(lastIndex)}</span>);
    }

    return <>{parts}</>;
}

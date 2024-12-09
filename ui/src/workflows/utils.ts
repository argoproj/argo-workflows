export function escapeInvalidMarkdown(markdown: string) {
    return markdown
        .replace(/([-*])\s(.*\n*)/g, '\\$1 $2') // escape list items
        .replace(/(\d)\.\s(.*\n*)/g, '$1\\. $2') // escape ordered list items
        .replace(/(-#)\s/g, '') // remove subheaders
        .replace(/\n/g, ' ') // remove line breaks
        .replace(/`{3}/g, '') // remove code blocks
        .replace(/^#+/g, '') // remove headers
        .replace(/^>+/g, '') // remove blockquotes
        .trim();
}

export function shortNodeName(node: {name: string; displayName: string}): string {
    return node.displayName || node.name;
}

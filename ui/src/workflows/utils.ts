export function escapeInvalidMarkdown(markdown: string) {
    return markdown
        .replace(/([-*])\s(.*\n*)/g, '\\$1 $2')
        .replace(/\n/g, ' ')
        .trim()
        .replace(/`{3}/g, '')
        .replace(/^#/g, '\\#')
        .replace(/^>/g, '\\>');
}

export function shortNodeName(node: {name: string; displayName: string}): string {
    return node.displayName || node.name;
}

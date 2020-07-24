export function getCollapsedNodeName(parent: string, numHidden: number): string {
    return '@collapsed/' + parent + '/' + numHidden;
}

export function isCollapsedNode(id: string): boolean {
    return id.startsWith('@collapsed/');
}

export function getCollapsedNodeParent(id: string): string {
    const split = id.split('/');
    return split[1];
}

export function getCollapsedNumHidden(id: string): number {
    const split = id.split('/');
    return Number(split[2]);
}

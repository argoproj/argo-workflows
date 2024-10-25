export function majorMinor(version: string) {
    if (version.includes('.')) {
        return version.substring(0, version.lastIndexOf('.'));
    }
    return version;
}

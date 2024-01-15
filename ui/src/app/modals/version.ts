export function majorMinor(version: string) {
    if (version.includes('.')) {
        return version.substring(0, version.lastIndexOf('.') - 1);
    }
    return version;
}

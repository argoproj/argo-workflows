export const majorMinor = (version: string) => (version.includes('.') ? version.substr(0, version.lastIndexOf('.')) : version);

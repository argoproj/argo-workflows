export const recent = (x: Date): boolean => {
    if (!x) {
        return false;
    }
    const minutesAgo = (new Date().getTime() - new Date(x).getTime()) / (1000 * 60);
    return minutesAgo < 15;
};

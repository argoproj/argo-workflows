export default function debounce<T extends (...args: any[]) => any>(fn: T, debounceMs: number) {
    let timer: number | null = null;

    const cancel = () => {
        if (timer !== null) {
            clearTimeout(timer);
            timer = null;
        }
    };

    const debouncedFn = (...args: Parameters<T>) => {
        cancel();

        timer = window.setTimeout(() => {
            fn(...args);
            timer = null;
        }, debounceMs);
    };

    return [debouncedFn, cancel];
}

export class MathOperations {
    static roundToTwoDigits(value: any): number {
        return typeof value === 'string' ? parseFloat(value).toFixed(2) : value.toFixed(2);
    }

    static roundTo(value: any, digits = 2): number {
        return typeof value === 'string' ? parseFloat(value).toFixed(digits) : value.toFixed(digits);
    }
}

// This shifter is used to shift the index of the first relevant item one index to the right, and subsequent one index to the left.
//
// Example:
//
// Shift: 0 1 2 3 4 ... to 0 1 3 2 4 ...
//
// shifter.get(0) -> 0
// shifter.get(1) -> 1
// shifter.start()
// shifter.get(2) -> 3
// shifter.get(3) -> 2
// shifter.get(4) -> 4
// ...
export class Shifter {
    private shifts: number;
    constructor() {
        this.shifts = -1;
    }

    public start(): void {
        if (this.shifts !== -1) {
            return;
        }
        this.shifts = 0;
    }

    public get(i: number): number {
        if (this.shifts === -1) {
            return i;
        } else if (this.shifts === 0) {
            this.shifts++;
            return i + 1;
        } else if (this.shifts === 1) {
            this.shifts = -1;
            return i - 1;
        }
        return i;
    }
}

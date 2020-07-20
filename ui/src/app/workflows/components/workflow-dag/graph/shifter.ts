export class Shifter {
    private shifts: number;
    constructor() {
        this.shifts = -1;
    }

    public startShift(): void {
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
            return i + 2;
        } else if (this.shifts === 1) {
            this.shifts++;
            return i - 1;
        } else if (this.shifts === 2) {
            this.shifts = -1;
            return i - 1;
        }
        return i;
    }
}

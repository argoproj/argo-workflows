// This shifter is used to shift the index of the first relevant item one index to the right, and subsequent one index to the left.
//
// A collapsed node always gets ordered before the other children: C 1 2, C is the collapsed node and 1, 2 are child nodes.
// What we want is 1 C 2. Since we must "stream" the ordering due to the way the graph code works at the moment, this class
// serves to shift the index of C and 1 when so desired. It will essentially map 0 -> 1, 1 -> 0, 2 -> 2 for ordering purposes.
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

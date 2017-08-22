import { PerfData } from './perf-data';

export class BuildPerformance {
    delay: PerfData[] = [];
    status: number = 0;
    throughput: PerfData[] = [];
}

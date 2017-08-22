import { Component, OnInit } from '@angular/core';

import { SortOperations } from '../../common/sortOperations/sortOperations';
import { SystemStatus, BuildPerformance } from '../../model';
import { PerformanceService, StatusService } from '../../services';
import { HasLayoutSettings, LayoutSettings } from '../layout/layout.component';

class LineChartSeries {
    key: string = '';
    values: string[] = [];
}

class BoxPlotChartSeries {
    values: {
        label: number,
        values: {
            Q1: number,
            Q2: number,
            Q3: number
        }
    }[];
}

@Component({
    templateUrl: './performance.html',
    styles: [ require('./performance.scss') ],
})
export class PerformanceComponent implements OnInit, HasLayoutSettings {
    public status: SystemStatus;
    public buildsLineChartSeries: LineChartSeries[];
    public buildsBoxPlotChartSeries: BoxPlotChartSeries[];
    public buildPerformance: BuildPerformance[] = [];

    constructor(private performanceService: PerformanceService, private statusService: StatusService) {
    }

    ngOnInit() {
        this.getStatusAsync();
        this.getBuilds();
    }

    get layoutSettings(): LayoutSettings {
        return {
            pageTitle: 'Performance'
        };
    }

    getStatusAsync() {
        this.statusService.getStatusAsync().subscribe(
            success => {
                this.status = success;
        });
    }

    getBuilds() {
        this.performanceService.getBuildPerformanceAsync().subscribe((results) => {
            this.buildPerformance = results;
            if (this.buildPerformance || this.buildPerformance.length) {
                this.buildsLineChartSeries = this.mapToLineChart(this.buildPerformance['throughput']);
                this.buildsBoxPlotChartSeries = this.mapToBoxPlotChart(this.buildPerformance['delay']);
            }
        });
    }

    mapToLineChart(data): LineChartSeries[] {
        data = SortOperations.sortBy(data, 'time');

        return [{
            'key': 'Data',
            'values': data.map(item => {
                return [
                    item.time * 1000, // to milliseconds
                    item.data === undefined ? 0 : item.data
                ];
            })
        }];
    }

    mapToBoxPlotChart(data): BoxPlotChartSeries[] {
        data = SortOperations.sortBy(data, 'time');

        return [
            {
                'values': data.map(item => {
                    return {
                        label: item.time * 1000, // to milliseconds
                        values: {
                            'Q1': item.min,
                            'Q2': item.data,
                            'Q3': item.max
                        }
                    };
                })
            }
        ];
    }
}

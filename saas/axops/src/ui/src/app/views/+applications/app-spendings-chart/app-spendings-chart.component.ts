import * as _ from 'lodash';
import * as d3 from 'd3';
import { Component, Input, ElementRef, OnDestroy, Renderer } from '@angular/core';
import { Observable } from 'rxjs';
import * as moment from 'moment';

import { SortOperations } from '../../../common/sortOperations/sortOperations';
import { SpendingsChartInput, ChartConfig, DateInterval, Utils } from '../../../common/chartSpendings';
import { ChartModificators } from '../../../common/chartSpendings/chartModificators';
import { PerfData, PerfBreakDownData } from '../../../model';
import { PerfDataService } from '../../../services';

declare let nv: any;
let nextId = 0;

@Component({
    selector: 'ax-app-spendings-chart',
    templateUrl: 'app-spendings-chart.html',
    styles: [ require('./app-spendings-chart.scss') ],
})
export class AppSpendingsChartComponent implements OnDestroy {
    @Input()
    dateFormatter: (input: number) => string;

    private data: SpendingsChartInput;
    private _appName: string;

    @Input()
    public deploymentName: string;

    @Input()
    public showTitle: boolean = false;

    @Input()
    public showTimeRange: boolean = true;

    @Input()
    set appName(appName: string) {
        this._appName = appName;
        this.loadSpendings(this.currentInterval.step.seconds,
            this.currentInterval.startTime,
            this.currentInterval.endTime).subscribe(results => {
            this.data = {perfData: results, interval: this.currentInterval};
            this.chartLoading = false;
            this.chartDataAvailable = true;
            let {chart, disposeChart} = this.createChart();
            this.chart = chart;
            this.disposeChart = disposeChart;
            this.svgEl = $(this.el.nativeElement).find('svg')[0];
            this.addUniqueClass();
            this.draw();
            this.customChartStyle();
            this.initEventHandlers();
        });
    }

    private currentInterval: DateInterval;
    private chartLoading: boolean = true;
    private selectedStepInfo: {left: number, width: number, height: number, text: string};
    private stepIndex: number;
    private preSelectionLeft: number;
    private preSelectionHeight: number;

    private chartDataAvailable: boolean = false;
    private currentSpendingChartClass: string;
    private svgEl: any;
    private disposeChart: any;
    private chart: any;

    private chartConfig: ChartConfig = {
        margin: {
            'right': 120,
            'left': 0,
            'top': 20,
            'bottom': 0
        },
        detailsColor: '#e5e5e5'
    };

    constructor(private el: ElementRef, private renderer: Renderer, private perfDataService: PerfDataService) {
        this.currentInterval = new DateInterval(30, moment().endOf('day'));
        this.dateFormatter = this.currentInterval.step.dateFormatter;
    }

    ngOnDestroy() {
        if (this.disposeChart) {
            this.disposeChart();
        }
    }

    addUniqueClass() {
        this.renderer.setElementClass(this.svgEl, `spendings-chart-${nextId}`, true);
        this.currentSpendingChartClass = `spendings-chart-${nextId}`;
        nextId++;
    }

    private draw() {
        if (this.svgEl && this.data) {
            this.drawChart(this.getSeries(this.data));
        }
    }

    private customChartStyle() {
        ChartModificators.addXLineTicks(d3, this.currentSpendingChartClass, 20, this.chartConfig, 1, 30);
        ChartModificators.hideXLineTicks(d3, this.currentSpendingChartClass, this.chartConfig);
        ChartModificators.transformXTicks(d3, this.currentSpendingChartClass, 25, 10, this.chartConfig);
    }

    private toBar(values: any[]) {
        let barValues = [];
        values.forEach((item, i) => {
            barValues.push(values[i]);
            barValues.push([
                values[i][0] + this.data.interval.step.seconds * 1000,
                values[i][1],
            ]);
        });
        return barValues;
    }

    private getSeries(input: SpendingsChartInput): any[] {
        let data = input.perfData.slice(0);
        // mock some vaules

        data = SortOperations.sortBy(data, 'time');
        let series = [{
            key: 'Spending',
            color: '#98dce7',
            strokeWidth: 3,
            values: this.toBar(data.map(item => {
                return [
                    item.time * 1000,
                    item.data === undefined ? 0 : Math.round(item.data) / 100
                ];
            })),
            area: true
        }];
        return series;
    }

    private drawChart(series: any[]) {
        let ticksList = _.map(series[0].values, v => v[0]);
        let maxValue = _.maxBy(series[0].values, v => v[1]);
        this.chart.forceX(ticksList);
        this.chart.height(170);
        this.chart.xAxis
            .tickValues(ticksList)
            .tickFormat((d, i) => {
                let dateFormatter = this.dateFormatter || (input => input.toString());
                return dateFormatter(d);
            });
        this.chart.forceY([0, maxValue[1] < 0.01 ? 1 : maxValue[1] * 1.1]);
        d3.select(this.svgEl)
            .datum(series)
            .transition()
            .duration(500)
            .call(this.chart);
    }

    private createChart() {
        let chart = nv.models.lineChart()
            .interpolate('basis')
            .x(function (d) {
                return d[0];
            })
            .y(function (d) {
                return d[1];
            })
            .useInteractiveGuideline(false)
            .clipEdge(false)
            .showXAxis(true)
            .showYAxis(false)
            .showLegend(false)
            .forceY([0])
            .margin(this.chartConfig.margin);
        chart.xAxis.showMaxMin(false);
        chart.yAxis
            .showMaxMin(true)
            .tickFormat(function (d, i) {
                // display thick only for max and min value
                return i ? '' : (d).toFixed(2);
            });

        chart.tooltip.enabled(false);
        chart.color(['#0AA755']);

        let {clear} = nv.utils.windowResize(() => {
            if (chart.update) {
                chart.update();
                this.customChartStyle();
            }
        });

        return {
            chart: chart,
            disposeChart: clear
        };
    }

    private loadSpendings(interval: number, startTime: number, endTime?: number): Observable<PerfData[]> {
        this.chartLoading = true;
        let type = this.deploymentName ? 'service' : 'app';

        return Observable.fromPromise(
            Promise.all<any>([
                this.perfDataService.getSpendingsBreakDownBy({
                    name: type,
                    interval,
                    startTime,
                    endTime,
                    filterBy: null
                }, true).toPromise(),
                this.perfDataService.getSpendings({interval, startTime, endTime}, true).toPromise()
            ]).then(result => {
                // Contains overall spendings for whole system. Rendered as line on top chart on the page
                let breakDownData: PerfBreakDownData[] = result[0];
                // Contains selected user spendings. Rendered as stack line on top chart on the page
                let perfData: PerfData[] = result[1];
                let timeToUtilization = new Map<number, number>();
                breakDownData.filter(item => {
                    return item.name === (type === 'app' ? this._appName : this.deploymentName);
                }).forEach(item => timeToUtilization.set(item.time, item.data));
                perfData.forEach(item => {
                    let roundedStepTime = moment.unix(item.time).startOf(this.currentInterval.step.unitOfTime).unix();
                    item.data = timeToUtilization.get(roundedStepTime) || 0;
                });
                return perfData;
            }));
    }

    public getTotalCost(): number {
        return this.data.perfData.map(a => a.data).reduce((a, b) => a + b, 0);
    }

    public getDateRangeDes(): string {
        if (this.currentInterval.durationDays === 1) {
            return this.currentInterval.step.rangeFormatter(moment().startOf('day'), moment().startOf('hour'));
        } else {
            return this.currentInterval.step.rangeFormatter(moment.unix(this.currentInterval.startTime),
                moment.unix(this.currentInterval.endTime));
        }
    }

    public getTooltipLeft(stepInfo): number {
        // calculate left tooltip position
        return Math.max(stepInfo.left, 26) + stepInfo.width / 2;
    }

    private initEventHandlers() {
        $(this.svgEl).on('mousemove', e => {
            let interval = this.data && Utils.getIntervalFromTime(
                    this.chart.lines.xScale().invert(e.offsetX - this.chart.margin().left) / 1000,
                    this.data.perfData,
                    this.data.interval.step.seconds, true);
            if (interval) {
                this.preSelectionLeft = this.chart.lines.xScale()(interval.left * 1000) + this.chart.margin().left + 1;
                this.preSelectionHeight = $(this.svgEl).height() - 30;
                this.selectedStepInfo = {
                    left: this.preSelectionLeft,
                    width: 30,
                    height: $(this.svgEl).height(),
                    text: ''
                };
                this.stepIndex = interval.index;
            } else {
                this.selectedStepInfo = null;
                this.stepIndex = -1;
            }
        });
    }
}

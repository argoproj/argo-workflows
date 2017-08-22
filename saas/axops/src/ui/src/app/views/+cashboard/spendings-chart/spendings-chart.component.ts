import * as _ from 'lodash';
import * as moment from 'moment';
import * as d3 from 'd3';
import { Component, Input, Output, EventEmitter, ElementRef, OnInit, OnDestroy, Renderer } from '@angular/core';

import { SortOperations } from '../../../common/sortOperations/sortOperations';
import { SpendingsChartInput, ChartConfig, Utils } from '../../../common/chartSpendings';
import { ChartModificators } from '../../../common/chartSpendings/chartModificators';
import { MathOperations } from '../../../common/mathOperations/mathOperations';

declare let nv: any;
let nextId = 0;

@Component({
    selector: 'ax-spendings-chart',
    templateUrl: './spendings-chart.html',
    styles: [ require('./spendings-chart.scss') ],
})
export class SpendingsChartComponent implements OnInit, OnDestroy {

    @Output()
    onStepClicked = new EventEmitter<number>();
    @Input()
    dateFormatter: (input: number) => string;
    @Input()
    utilizationOnlyViewMode = false;

    selectedStepInfo: {left: number, width: number, height: number, text: string};
    preSelectionLeft: number;
    preSelectionHeight: number;
    estimatedTooltip: {shown: boolean, day: number} = { shown: false, day: -1 };
    projectionLine: {left: number, top: number};

    private currentSpendingChartClass: string;
    private input: SpendingsChartInput;
    private svgEl: any;
    private chart: any;
    private disposeChart: any;
    private step: { startTime: number, endTime: number, index: number };
    private chartConfig: ChartConfig = {
        margin: {
            'right': 0,
            'left': 30,
            'top': 20,
            'bottom': 0
        },
        detailsColor: '#e5e5e5'
    };

    constructor(private el: ElementRef, private renderer: Renderer) {}

    ngOnInit() {
        let { chart, disposeChart } = this.createChart();
        this.chart = chart;
        this.disposeChart = disposeChart;
        this.addUniqueClass();
        this.svgEl = $(this.el.nativeElement).find('svg')[0];
        this.draw();
        if (!this.utilizationOnlyViewMode) {
            this.initEventHandlers();
        }
    }

    ngOnDestroy() {
        if (this.disposeChart) {
            this.disposeChart();
        }
        nextId = 0;
    }

    @Input()
    set data(input: SpendingsChartInput) {
        this.input = input;
        this.draw();
        this.initializeProjectionLine();
    }

    get data(): SpendingsChartInput {
        return this.input;
    }

    @Input()
    set hightlightedStep(value: {startTime: number, endTime: number, index: number}) {
        if (value) {
            this.highlightStep(value.startTime, value.endTime, value.index);
        }
    }

    addUniqueClass() {
        this.renderer.setElementClass(this.el.nativeElement.children[0], `spendings-chart-${nextId}`, true);
        this.currentSpendingChartClass = `spendings-chart-${nextId}`;
        nextId++;
    }

    highlightStep(startTime: number, endTime: number, index: number) {
        this.step = {startTime: startTime, endTime: endTime, index: index};
        if (this.chart) {
            this.customChartStyle();
            if (index > -1) {
                let left = this.chart.xAxis.scale()(startTime * 1000) + this.chart.margin().left;
                let right = this.chart.xAxis.scale()(endTime * 1000) + this.chart.margin().left;
                let startTimeFormatted = this.input.interval.step.dateFormatter(startTime * 1000);
                let endTimeFormatted = this.input.interval.step.dateFormatter(endTime * 1000);
                this.selectedStepInfo = {
                    left: left,
                    width: right - left,
                    height: $(this.svgEl).height(),
                    text: `${startTimeFormatted} - ${endTimeFormatted}`
                };
            } else {
                this.selectedStepInfo = null;
            }
        }
    }

    notifyStepClicked(index: number) {
        this.onStepClicked.emit(index);
    }

    private initializeProjectionLine() {
        if (this.input && this.input.interval.isCurrentDay && this.input.perfData.length > 0 && this.chart) {
            let totalSpend = 0;
            this.input.perfData.forEach(item => {
                totalSpend += item.data;
            });
            this.estimatedTooltip.day = MathOperations.roundToTwoDigits(24 / moment().hour() * totalSpend);
            let perf = this.input.perfData[0];
            this.projectionLine = {
                left: this.chart.lines.xScale()(perf.time * 1000 + this.input.interval.step.seconds * 1000) + this.chart.margin().left,
                top: this.chart.lines.yScale()(perf.data / 100) + this.chart.margin().top
            };
        } else {
            this.estimatedTooltip = { shown: false, day: -1 };
            this.projectionLine = null;
        }
    }

    private draw() {
        if (this.svgEl && this.input) {
            this.drawChart(this.getSeries(this.input), this.input.interval.isCurrentDay);
        }
        if (this.step) {
            this.highlightStep(this.step.startTime, this.step.endTime, this.step.index);
        }
    }

    private initEventHandlers() {
        $(this.svgEl).on('mousemove', e => {
            let interval = this.input && Utils.getIntervalFromTime(
                this.chart.lines.xScale().invert(e.offsetX - this.chart.margin().left) / 1000,
                this.input.perfData,
                this.input.interval.step.seconds);
            if (interval) {
                this.preSelectionLeft = this.chart.lines.xScale()(interval.left * 1000) + this.chart.margin().left;
                this.preSelectionHeight = $(this.svgEl).height();
            }
            if (this.selectedStepInfo) {
                this.estimatedTooltip.shown = e.offsetX > this.selectedStepInfo.width * this.input.perfData.length;
            }
        });
        $(this.svgEl).on('click', e => {
            let interval = this.input && Utils.getIntervalFromTime(
                this.chart.lines.xScale().invert(e.offsetX - this.chart.margin().left) / 1000,
                this.input.perfData,
                this.input.interval.step.seconds);
            this.notifyStepClicked(interval ? interval.index : -1);
        });
    }

    private customChartStyle() {
        ChartModificators.addXLineTicks(d3, this.currentSpendingChartClass, 6, this.chartConfig);
        ChartModificators.addYLineTicks(d3, this.currentSpendingChartClass, 30, -30, 0, this.chartConfig);
        ChartModificators.transformXTicks(d3, this.currentSpendingChartClass, 20, -20, this.chartConfig);
        ChartModificators.transformValue(d3, this.currentSpendingChartClass, 'nv-axisMin-y', 0, -10, this.chartConfig);
        ChartModificators.transformValue(d3, this.currentSpendingChartClass, 'nv-axisMax-y', 0, -10, this.chartConfig);
    }

    private toBar(values: any[]) {
        let barValues = [];
        values.forEach((item, i) => {
            barValues.push(values[i]);
            barValues.push([
                values[i][0] + this.input.interval.step.seconds * 1000,
                values[i][1],
            ]);
        });
        return barValues;
    }

    private getSeries(input: SpendingsChartInput): any[] {
        let data = input.perfData.slice(0);
        data = SortOperations.sortBy(data, 'time');
        let series = [{
            key: 'Utilization',
            color: '#98dce7',
            strokeWidth: 6,
            values: this.toBar(data.map(item => {
                return [
                    item.time * 1000,
                    item.max === undefined ? 0 : Math.round(item.max) / 100
                ];
            })),
            area: !this.utilizationOnlyViewMode
        }];
        if (!this.utilizationOnlyViewMode) {
            series.push({
                key: 'Spending',
                color: '#FFFFFF',
                strokeWidth: 3,
                values: this.toBar(data.map(item => {
                    return [
                        item.time * 1000,
                        item.data === undefined ? 0 : Math.round(item.data) / 100
                    ];
                })),
                area: false
            });
        }
        return series;
    }

    private drawChart(series: any[], singleDayMode: boolean) {
        let ticksList = _.map(series[0].values, v => v[0]);
        this.chart.forceX(
            singleDayMode ? [ticksList[0], moment(this.input.interval.startTime * 1000).endOf('day').unix() * 1000] : ticksList);
        this.chart.xAxis
            .tickValues(ticksList)
            .tickFormat((d, i) => {
                let dateFormatter = this.dateFormatter || (input => input.toString());
                return (i % 2 && i < ticksList.length - 1) ? '' : dateFormatter(d);
            });
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
            .showXAxis(!this.utilizationOnlyViewMode)
            .showYAxis(!this.utilizationOnlyViewMode)
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

        let { clear } = nv.utils.windowResize(() => {
            if (chart.update) {
                chart.update();
            }

            if (this.step != null) {
                this.highlightStep(this.step.startTime, this.step.endTime, this.step.index);
            }
        });

        return {
            chart: chart,
            disposeChart: clear
        };
    }
}

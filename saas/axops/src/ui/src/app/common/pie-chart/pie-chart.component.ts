import * as d3 from 'd3';
import { Component, Input, ElementRef } from '@angular/core';

import { PieChartInput } from './pie-chart.view-models';

declare let nv: any;


@Component({
    selector: 'ax-pie-chart',
    templateUrl: 'pie-chart.html',
    styles: [ require('./pie-chart.scss') ],
})
export class PieChartComponent {
    private svgEl: any;
    private chart: any;
    private _data: PieChartInput[];

    @Input()
    set data(data: PieChartInput[]) {
        this._data = data;

        this.chart = this.createChart();
        this.svgEl = $(this.el.nativeElement).find('svg')[0];

        d3.select(this.svgEl)
            .datum(this._data)
            .transition().duration(350)
            .call(this.chart);
    }

    constructor(private el: ElementRef) {
    }

    private createChart() {
        let chart = nv.models.pieChart()
            .x(function (d) {
                return d.label;
            })
            .y(function (d) {
                return d.value;
            })
            .showLabels(false)    // Display pie labels
            .labelThreshold(.05)  // Configure the minimum slice size for labels to show up
            .labelType('percent') // Configure what type of data to show in the label. Can be "key", "value" or "percent"
            .donut(true)          // Turn on Donut mode.
            .donutRatio(0.87)     // Configure how big you want the donut hole size to be.
            .color(function (d) {
                return d.color;
            })
            .width(130)
            .height(130)
            .showLegend(false)
            .growOnHover(false)
            .margin({
                'right': 0,
                'left': 0,
                'top': 0,
                'bottom': 0
            })
            .noData('');

        chart.tooltip.enabled(false);

        return chart;
    }
}

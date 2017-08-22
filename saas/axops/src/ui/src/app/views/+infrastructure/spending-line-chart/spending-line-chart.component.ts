import * as _ from 'lodash';
import * as d3 from 'd3';
import {Component, Input} from '@angular/core';
import {MathOperations} from '../../../common/mathOperations/mathOperations';
import {TimeFormatter} from '../../../common/timeFormatter/timeFormatter';
import {TranslateService} from 'ng2-translate/ng2-translate';

declare let nv: any;

@Component({
    selector: 'ax-spending-line-chart',
    templateUrl: './spending-line-chart.html',
    styles: [ require('./spending-line-chart.scss') ],
})

export class SpendingLineChartComponent {

    @Input()
    set data(value) {
        this.drawChart(value);
    }

    constructor(private translateService: TranslateService) {
    }

    drawChart(series: Object) {
        if (series) {
            let that = this;

            let maxSpendValue = _.maxBy(series[0]['values'], (v) => {
                return v[1];
            });

            let ticksList = _.map(series[0].values, (v) => {
                return v[0];
            });

            nv.addGraph(function () {
                let chart = nv.models.lineChart()
                    .useInteractiveGuideline(true)
                    .interactive(true)
                    .x(function (d) {
                        return d[0];
                    })
                    .y(function (d) {
                        return d[1];
                    })
                    .showLegend(false)
                    .forceY([0, maxSpendValue[1] * 1.4]) // round to thousand with 40% top margin from highest value
                    .forceX(ticksList)
                    .clipEdge(false)
                    .margin({ 'right': 70, 'left': 70 });
                chart.xAxis
                    .showMaxMin(false)
                    .axisLabel(that.translateService.get('Date')['value'])
                    .tickValues(ticksList)
                    .tickFormat(function (d, i) {
                        return (i % 4 && i < ticksList.length - 1) ? '' : TimeFormatter.monthAndDay(TimeFormatter.toUtc(d));
                    });
                chart.yAxis
                    .showMaxMin(false)
                    .tickFormat(function (d) {
                        return '$' + (d).toFixed(2);
                    });
                chart.interactiveLayer.tooltip.contentGenerator(function (d) {
                    return '<div>' +
                        '<p>' + TimeFormatter.onlyDate(TimeFormatter.toUtc(d.value)) + '</p>' +
                        '<p>' + that.translateService.get('Builds')['value'] + ': $' +
                        MathOperations.roundToTwoDigits(d.series[0].value) + '</p>' + '</div>';
                });
                chart.color(['#0AA755']);

                d3.select('#spending-line svg')
                    .datum(series)
                    .transition()
                    .duration(500)
                    .call(chart);

                // Aria requirement and fix for chart library. The problem was elements with the same id's
                chart.dispatch.on('renderEnd', function () {
                    $.each($('clipPath'), (index, value) => {
                        $(value).attr('id', $(value).attr('id') + index);
                    });
                });
                chart.dispatch.renderEnd();

                nv.utils.windowResize(chart.update);

                return chart;
            });
        }
    }
}

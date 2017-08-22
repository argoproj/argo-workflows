import * as _ from 'lodash';
import * as d3 from 'd3';
import {Component, Input} from '@angular/core';

import {TimeFormatter} from '../../../common/timeFormatter/timeFormatter';
import {TranslateService} from 'ng2-translate/ng2-translate';

declare let nv: any;

@Component({
    selector: 'ax-builds-line-chart',
    templateUrl: './builds-line-chart.html',
    styles: [ require('./builds-line-chart.scss') ],
})
export class BuildsLineChartComponent {
    constructor(private translateService: TranslateService) {
    }

    @Input()
    set data(value){
        this.drawChart(value);
    }

    drawChart(series: Object) {
        if (series) {
            let that = this;
            let ticksList = _.filter(_.map(series[0].values, (v) => {
                return v[0];
            }), function(item, index) {
                return index % 5 === 0; // return every fifth element
            });

            nv.addGraph(function() {
                let chart = nv.models.lineChart()
                    .useInteractiveGuideline(true)
                    .interactive(false)
                    .useVoronoi(false)
                    .x(function(d) { return d[0]; })
                    .y(function(d) { return d[1]; })
                    .showLegend(false)
                    .forceY([0])
                    .clipEdge(false)
                    .margin({'left': 80, 'right': 60});
                chart.xAxis
                    .showMaxMin(true)
                    .tickValues(ticksList)
                    .tickFormat(function(d, i){
                        return TimeFormatter.monthAndDay(TimeFormatter.toUtc(d));
                    });
                chart.yAxis
                    .tickFormat(function(d) {
                        return d;
                    });
                chart.interactiveLayer.tooltip.contentGenerator(function(d){
                    return '<div>' +
                        '<p>' + TimeFormatter.onlyDate(TimeFormatter.toUtc(d.value)) + '</p>' +
                        '<p>' + that.translateService.get('Jobs')['value'] + ': ' + d.series[0].value + '</p>' +
                        '</div>';
                });
                chart.color(['#0AA755']);

                d3.select('#builds-line svg')
                    .datum(series)
                    .transition()
                    .duration(500)
                    .call(chart);

                // Aria requirement and fix for chart library. The problem was elements with the same id's
                chart.dispatch.on('renderEnd', function(){
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

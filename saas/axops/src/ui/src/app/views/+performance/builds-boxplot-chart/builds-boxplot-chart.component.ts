import * as d3 from 'd3';
import * as moment from 'moment';
import * as _ from 'lodash';
import {Component, Input} from '@angular/core';
import {TimeFormatter} from '../../../common/timeFormatter/timeFormatter';
import {MathOperations} from '../../../common/mathOperations/mathOperations';
import {TranslateService} from 'ng2-translate/ng2-translate';

declare var nv: any;

@Component({
    selector: 'ax-builds-boxplot-chart',
    templateUrl: './builds-boxplot-chart.html',
    styles: [ require('./builds-boxplot-chart.scss') ],
})
export class BuildsBoxPlotChartComponent {

    constructor(private translateService: TranslateService) {
    }

    @Input()
    set data(value){
        this.drawChart(value);
    }

    private formatTime(durationSeconds) {
        let momentTimeStart = moment.utc(0);
        let momentTime = moment.utc(durationSeconds * 1000);
        let duration = moment.duration(momentTime.diff(momentTimeStart));

        if (momentTime.diff(momentTimeStart, 'seconds') === 0) {
            return MathOperations.roundTo(durationSeconds, 2) + ' s';
        }

        if (momentTime.diff(momentTimeStart, 'hours') === 0) {
            return ('0' + duration.minutes()).slice(-2) + ':' + ('0' + duration.seconds()).slice(-2) + ' m';
        }

        return ('0' + momentTime.diff(momentTimeStart, 'hours')).slice(-2) + ':' + ('0' + duration.minutes()).slice(-2) + ' h';
    }

    private drawChart(series: Object) {
        if (series) {
            let ticksList = _.filter(_.map(series[0].values,
                v => v['label']),
                (item, index) => index % 5 === 0 // return every fifth element
            );

            nv.addGraph(() => {
                let chart = nv.models.boxPlotChart()
                    .x(d => d.label)
                    .y(d => d.values.Q3)
                    .color(['#0AA755'])
                    .margin({'left': 80, 'right': 50, 'top': 25});
                chart.xAxis
                    .showMaxMin(true)
                    .tickValues(ticksList)
                    .tickFormat((d, i) => TimeFormatter.monthAndDay(TimeFormatter.toUtc(d)));

                chart.yAxis.tickFormat((d, i) => this.formatTime(d));

                chart.tooltip.contentGenerator((d) => {
                    return '<div>' +
                        '<p>' + TimeFormatter.onlyDate(TimeFormatter.toUtc(d.value)) + '</p>' +
                        '<p>' + this.translateService.get('90%')['value'] +
                        ': ' + this.formatTime(d.data.values.Q3) + '</p>' +
                        '<p>' + this.translateService.get('median')['value'] +
                        ': ' + this.formatTime(d.data.values.Q2) + '</p>' +
                        '<p>' + this.translateService.get('10%')['value'] +
                        ': ' + this.formatTime(d.data.values.Q1) + '</p>' +
                        '</div>';
                });
                chart.color(['#0AA755']);

                d3.select('#builds-boxplot svg')
                    .datum(series[0].values)
                    .call(chart);

                // Aria requirement and fix for chart library. The problem was elements with the opacity equal 0
                chart.dispatch.on('renderEnd', () => {
                    $.each($('text'), (index, value) => {
                        if ($(value).attr('opacity') === '0') {
                            $(value).remove();
                        }
                    });
                });
                chart.dispatch.renderEnd();

                nv.utils.windowResize(chart.update);
                return chart;
            });
        }
    }
}

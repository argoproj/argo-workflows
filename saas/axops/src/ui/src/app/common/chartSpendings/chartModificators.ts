import { ChartConfig } from './spendingChart.view-models';

export class ChartModificators {
    static addXLineTicks(d3, chartClass: string, tickHeight = 0, chartConfig?, translateX = 0, translateY = 0) {
        if (!chartConfig) {
            chartConfig = new ChartConfig();
        }
        // add x line ticks
        if (!d3.selectAll(`.${chartClass} .nv-x .tick line.short-x-tick`)[0].length) {
            d3.selectAll(`.${chartClass} .nv-x .tick`)
                .append('line')
                .attr('class', 'short-x-tick')
                .attr('stroke-width', 2)
                .attr('y2', `-${tickHeight}`)
                .attr('transform', `translate(${translateX}, ${translateY})`)
                .style('stroke', chartConfig.detailsColor);
        }
    }

    static hideXLineTicks(d3, chartClass: string, chartConfig?) {
        if (!chartConfig) {
            chartConfig = new ChartConfig();
        }
        // hide x line ticks
        let tickLength = d3.selectAll(`.${chartClass} .nv-x .tick line.short-x-tick`)[0].length;
        if (tickLength > 15) {
            d3.selectAll(`.${chartClass} .nv-x .tick`)
                .each(function (d, i) {
                    if (i % 2 && i > 0 && i < tickLength - 1) {
                        d3.select(this).attr('visibility', 'hidden');
                    }
                });
        }
    }

    static addYLineTicks(d3, chartClass: string, tickWidth = 0, translateX = 0, translateY = 0, chartConfig?) {
        if (!chartConfig) {
            chartConfig = new ChartConfig();
        }
        // add y line ticks
        if (!d3.selectAll(`.${chartClass} .nv-y .tick line.short-y-tick`)[0].length) {
            d3.selectAll(`.${chartClass}  .nv-y .tick`)
                .append('line')
                .attr('class', 'short-y-tick')
                .attr('x2', `${tickWidth}`)
                .attr('transform', `translate(${translateX}, ${translateY})`)
                .style('stroke', chartConfig.detailsColor);
        }
    }

    static transformXTicks(d3, chartClass: string, translateX: number, translateY: number, chartConfig?) {
        if (!chartConfig) {
            chartConfig = new ChartConfig();
        }
        // move x ticks value above 0 line
        d3.selectAll(`.${chartClass} .nv-x .tick text`)
            .attr('transform', `translate(${translateX}, ${translateY})`)
            .style('font-size', '13')
            .style('font-weight', '100')
            .style('fill', chartConfig.detailsColor);
    }

    static addTitleToTopLine(d3, chartClass: string, translateX: number, translateY: number, text: string, chartConfig? ) {
        if (!chartConfig) {
            chartConfig = new ChartConfig();
        }
        if (!d3.select(`.${chartClass} .nv-axisMax-y .top-y-line text`)[0][0]) {
            // add title to top dashed line
            d3.select(`.${chartClass} .nv-axisMax-y`)
                .append('text')
                .attr('transform', `translate(${translateX}, ${translateY})`)
                .style('font-size', '13')
                .style('font-weight', '100')
                .style('fill', chartConfig.detailsColor)
                .text(`${text}`);
        }
    }

    static transformValue(d3, chartClass: string, axisClass: string, translateX: number, translateY: number, chartConfig?) {
        if (!chartConfig) {
            chartConfig = new ChartConfig();
        }
        // move max y value above dashed line
        d3.select(`.${chartClass} .${axisClass} text`)
            .attr('transform', `translate(${translateX}, ${translateY})`)
            .style('font-size', '13')
            .style('font-weight', '100')
            .style('fill', chartConfig.detailsColor);
    }

    static transformYTickValues(d3, chartClass: string, translateX: number, translateY: number, chartConfig?) {
        if (!chartConfig) {
            chartConfig = new ChartConfig();
        }
        // move y line ticks
        d3.selectAll(`.${chartClass} .nv-y .nv-axis .tick text`)
            .attr('transform', `translate(${translateX}, ${translateY})`)
            .style('font-size', '13')
            .style('font-weight', '100')
            .style('fill', chartConfig.detailsColor);
    }
}

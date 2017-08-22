import * as _ from 'lodash';
import {Component, Input} from '@angular/core';

import {SpendingsDetail} from '../../../model';
import {MathOperations} from '../../../common/mathOperations/mathOperations';

@Component({
    selector: 'ax-spending-square-chart',
    templateUrl: './spending-square-chart.html',
    styles: [ require('./spending-square-chart.scss') ],
})
export class SpendingSquareChartComponent {
    public charts: AppChart[];
    public strategies: IStrategy[];

    static stringChain(stringList: string[], length): string {
        return stringList.length >= length ? _.chain(stringList).take(length).value().join(', ') + ' (...)' : stringList[0];
    }

    constructor() {
        this.strategies = [
            new ServicesSortingStrategy(),
            new ServicesMergingStrategy(),
        ];
    }

    @Input()
    set data(value) {
        let apps = this.mapToApps(value);

        for (let strategy of this.strategies) {
            apps = strategy.Apply(apps);
        }

        let charts = _.map(apps, AppChart.Create);

        this.charts = _.sortBy(charts, (item) => {
            return Number(item.appTotalSpent);
        }).reverse();
    }

    mapToApps(value: SpendingsDetail[]): App[] {
        let masterGroups = _.groupBy(_.groupBy(value, x => x.cost_id.app + '#' + x.cost_id.service), x => x[0].cost_id.app);

        let groupedByAppThenByService = _.map(masterGroups, singleApp => {

            let services = _.map(singleApp, x => {
                return {
                    name: x[0].cost_id.service,
                    spent: _.sum(_.map(x, y => y.spent)) / 100,
                    displayedSpent: MathOperations.roundToTwoDigits(_.sum(_.map(x, y => y.spent)) / 100)
                };
            });

            return {
                name: singleApp[0][0].cost_id.app,
                services: services,
                totalSpent: MathOperations.roundToTwoDigits(_.sum(_.map(services, s => s.spent)))
            };
        });

        return groupedByAppThenByService;
    }
}

/// ======================== View Model classes ==============================
class AppChart {
    // values to show
    appName: string;
    appTotalSpent: number;
    boxes: Array<ServiceBox>;
    enableHover: boolean = true;
    static Create(app: App) {
        let chart = new AppChart();
        chart.appName = app.name;
        chart.appTotalSpent = MathOperations.roundToTwoDigits(app.totalSpent);
        chart.boxes = _.map(app.services, service => new ServiceBox(service));

        let calculator = new ChartSizeCalculator(_.map(app.services, x => x.spent));
        let height = calculator.getHeightPercents();
        let topWidth = calculator.GetTopWidthPercents();
        let bottomWidth = calculator.GetBottomWidthPercents();

        if (chart.boxes[0]) {
            chart.boxes[0].height = height;
            chart.boxes[0].width = topWidth;
        }
        if (chart.boxes[1]) {
            chart.boxes[1].height = height;
            chart.boxes[1].width = 100 - topWidth;
        }
        if (chart.boxes[2]) {
            chart.boxes[2].height = 100 - height;
            chart.boxes[2].width = bottomWidth;
        }
        if (chart.boxes[3]) {
            chart.boxes[3].height = 100 - height;
            chart.boxes[3].width = 100 - bottomWidth;
        }
        // This is a custom requirement to prevent overloading of internal info on price breakdown chart
        if (app.name === 'axsys') {
            chart.enableHover = false;
        }

        return chart;
    }
}

class App {
    name: string;
    services: Array<Service>;
    totalSpent: number;
}

class Service {
    name: string;
    spent: number;
    displayedSpent: number;
}

class Box {
    width: number;
    height: number;
}

class ServiceBox extends Box {
    service: Service;

    constructor(service: Service) {
        super();
        this.service = service;
    }
}

///======================Array transformations ===============================
interface IStrategy {
    Apply(apps: App[]): App[];
}

class ServicesSortingStrategy implements IStrategy {
    Apply(apps: App[]): App[] {
        apps.forEach((item, index) => {
            apps[index].services = _.sortBy(apps[index].services, x => x.spent).reverse();
        });
        return apps;
    }
}

class ServicesMergingStrategy implements IStrategy {
    Apply(apps: App[]): App[] {
        for (let index in apps) {
            if (apps[index].services.length < 4) {
                continue;
            }

            let services = apps[index].services;

            let bigOnes = _.pullAt(services, [0, 1, 2]);

            let merged = {
                name: SpendingSquareChartComponent.stringChain(_.map(services, x => x.name), 3),
                spent: _.sum(_.map(services, x => x.spent)),
                displayedSpent: MathOperations.roundToTwoDigits(_.sum(_.map(services, x => x.spent))),
                services: _.map(services, x => {
                    return {
                        name: x.name,
                        spent: x.spent
                    };
                })
            };

            bigOnes.push(merged);
            apps[index].services = bigOnes;
        }

        return apps;
    }
}

// ================ Square calculator==============
////      Algorithm
////
////                     y              (A-y)
////     -        ''''''''''''''''''''''''''''''''
////     |        '             '                '
////     |        '             '                '
////     |        '     a       '      b         '
////     |     x  '             '                '
////     |        '             '                '
////  A  |        '             '                '
////     |        ''''''''''''''''''''''''''''''''
////     |        '                  '           '
////     |        '                  '           '
////     |  (A-x) '         c        '     d     '
////     |        '                  '           '
////     |        '                  '           '
////     |        '                  '           '
////     -        ''''''''''''''''''''''''''''''''
////                      z              (A-z)
////
////              |------------------------------|
////                            A
////
//// x = (b+a)/A
////
//// y = a*A/(b+a)
////
//// z = c /(A - (b+a)/A)
////

class ChartSizeCalculator {
    private a: number;
    private b: number;
    private c: number;
    private d: number;
    private A: number;

    constructor(values: number[]) {
        this.a = values[0] ? values[0] : 0;
        this.b = values[1] ? values[1] : 0;
        this.c = values[2] ? values[2] : 0;
        this.d = values[3] ? values[3] : 0;

        this.A = Math.sqrt(this.a + this.b + this.c + this.d);
    }

    getHeightPercents() {
        if (this.A <= 0) {
            return 0;
        }

        let result = (this.b + this.a) / this.A;
        return this.toPercents(result, this.A);
    }

    GetTopWidthPercents() {
        if ((this.b + this.a) <= 0) {
            return 0;
        }

        let result = this.a * this.A / (this.b + this.a);
        return this.toPercents(result, this.A);
    }

    GetBottomWidthPercents() {
        if (this.A <= 0 || (this.A - (this.b + this.a) / this.A) <= 0) {
            return 0;
        }

        let result = this.c / (this.A - (this.b + this.a) / this.A);
        return this.toPercents(result, this.A);
    }

    toPercents(part: number, total: number) {
        if (total <= 0) {
            return 0;
        }
        return part * 100 / total;
    }
}


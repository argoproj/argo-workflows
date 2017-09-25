import { Pipe, PipeTransform } from '@angular/core';

const COLORS: string[][] = [
    ['#31a0cb', '#4fc09f'],
    ['#40ace4', '#b98ac2'],
    ['#4879e5', '#ade75d'],
    ['#7a6cc2', '#8ab3c2'],
    ['#c2b070', '#a44a8c'],
    ['#25d3c0', '#e4c772'],
    ['#e9d98e', '#da8b55'],
    ['#6e64b5', '#e791dc'],
    ['#4fcb79', '#057883'],
    ['#8cc0ba', '#3012a6'],
];

@Pipe({
    name: 'catalogAppBg',
})
export class CatalogAppBgPipe implements PipeTransform {
    transform(name: string) {
        return name && COLORS[(name[0].charCodeAt(0) + (name[2] && name[2].charCodeAt(0)) || 0) % 10] || ['', ''];
    }
}

import { Component, OnInit, Input } from '@angular/core';

@Component({
    selector: 'ax-loader-list-mockup',
    templateUrl: './loader-list-mockup.html',
    styles: [ require('./loader-list-mockup.scss') ],
})
export class LoaderListMockupComponent implements OnInit {
    public itemsList: any[] = [];

    @Input()
    public itemHeight: number = 60;

    @Input()
    public itemOpacity: number = 0.3;

    @Input()
    public marginLeft: number = 0;

    @Input()
    public marginRight: number = 0;

    @Input()
    public itemGap: number = 8;

    @Input()
    public itemsLength: number;

    @Input()
    public customClass: string;

    @Input()
    public noMarginTop: boolean = false;

    public ngOnInit() {
        if (!this.itemsLength) {
            // count how many loader template elements we need to fill full screen
            for (let i = 0; i < window.innerHeight / this.itemHeight; i++) { // 60 is height of single element
                this.itemsList.push('');
            }
        } else {
            for (let i = 0; i < this.itemsLength; i++) {
                this.itemsList.push('');
            }
        }
    }
}

import {Component, ElementRef, Inject, OnDestroy, EventEmitter} from '@angular/core';
import {LoaderService} from '../../services';

@Component({
    selector: 'ax-loader',
    templateUrl: './loader.html',
    styles: [ require('./loader.scss') ],
})

export class LoaderComponent implements OnDestroy {
    private streamShow: EventEmitter<any>;
    private streamHide: EventEmitter<any>;
    private calls: string[] = [];

    constructor(@Inject(LoaderService) private loaderService, @Inject(ElementRef) private el) {
        this.el.nativeElement.hidden = true;
        this.streamShow = this.loaderService.show.subscribe(value => {this.showLoader(value); } );
        this.streamHide = this.loaderService.hide.subscribe(value => {this.hideLoader(value); } );
    }

    ngOnDestroy() {
        this.streamHide.unsubscribe();
        this.streamShow.unsubscribe();
    }

    showLoader(call) {
        this.calls.push(call);
        this.el.nativeElement.hidden = false;
        $('body').addClass('overflow-hidden');
    }

    hideLoader(call) {
        let indexOfCall = this.calls.indexOf(call);
        if (indexOfCall !== -1) {
            this.calls.splice(indexOfCall, 1);
        }
        if (this.calls.length === 0) {
            this.el.nativeElement.hidden = true;
            $('body').removeClass('overflow-hidden');
        }
    }
}

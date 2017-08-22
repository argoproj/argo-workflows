import {Directive, ElementRef, Renderer, HostListener, Input} from '@angular/core';
import {MathOperations} from '../../../../common/mathOperations/mathOperations';

@Directive({
    selector: '[ax-box-tooltip]',
})
export class TooltipDirective {
    private tooltipEnabled: boolean = true;
    @Input('ax-box-tooltip')
    set box(box) {
        this.check(box);
    }

    @Input('ax-box-tooltip-enable')
    set enable(enable){
        this.tooltipEnabled = !!enable;
    }

    constructor(private el: ElementRef, private renderer: Renderer) {
    }

    @HostListener('mouseenter', ['$event'])
    showTooltip() {
        if (!this.tooltipEnabled) {
            return;
        }
        $(this.el.nativeElement).find('.tooltip__box').removeAttr('hidden');

        // move tooltip content to the middle
        let position = $(this.el.nativeElement).position();
        let tooltipContentHeight = $(this.el.nativeElement).find('.tooltip__content').height();
        $(this.el.nativeElement).find('.tooltip__content').css('margin-top',
            position.top >= (tooltipContentHeight / 2) ? (tooltipContentHeight / -2) : -position.top);
    }

    @HostListener('mouseleave', ['$event'])
    hideTooltip() {
         $(this.el.nativeElement).find('.tooltip__box').attr('hidden', 'hidden');
    }

    check(box) {
        // set label position on the chart box
        $(this.el.nativeElement).children('span').css('line-height', $(this.el.nativeElement).children('span').height() + 'px');

        // add toltip div
        this.renderer.createElement(this.el.nativeElement.children[0], 'div');

        // add class and attr
         $(this.el.nativeElement).find('div').addClass('tooltip__box')
             .attr('hidden', 'hidden').append('<div class="tooltip__arrow"></div>');

        // set tooltip position
        $(this.el.nativeElement).find('.tooltip__box').css('top', $(this.el.nativeElement.children[0]).height() / 2);

        // set tooltip content
        let template = `<div class="tooltip__content">
            <ul>
                ${this.getListToTooltip(box)}
            </ul>
        </div>`;
        $(this.el.nativeElement).find('.tooltip__box').append(template);
    }

    getListToTooltip(box) {
        let list = '';
        if (box.service.hasOwnProperty('services')) {
            box.service.services.forEach((v) => {
                list += MathOperations.roundTo(v.spent, 3) === 0 ? '' : '<li>' + v.name + ': $' + MathOperations.roundTo(v.spent, 3) +
                '</li>'; // don't display service if it's value  rounded to 3 digits is equal 0
            });
        } else {
            list = '<li>' + box.service.name + ': $' + MathOperations.roundToTwoDigits(box.service.spent) + '</li>';
        }

        return list;
    }
}

import { Directive, ElementRef, Renderer } from '@angular/core';

@Directive({
    selector: '[ax-autocomplete-off]'
})
export class AutocompleteOffDirective {
    constructor(el: ElementRef, renderer: Renderer) {
        renderer.setElementAttribute(el.nativeElement, 'readonly', 'true');
        renderer.listen(el.nativeElement, 'focus', () => {
            renderer.setElementAttribute(el.nativeElement, 'readonly', null);
        });
    }
}

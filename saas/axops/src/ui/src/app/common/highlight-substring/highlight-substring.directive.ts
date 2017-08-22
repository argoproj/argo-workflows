import { Directive, ElementRef, Input, AfterViewInit} from '@angular/core';

@Directive({
    selector: '[ax-highlight-substring]'
})

export class HighlightSubstringDirective implements AfterViewInit {
    @Input('ax-highlight-substring')
    highlightSubstring = '';
    @Input('ax-original-value')
    originalValue = ''; // if you use pipe and displayed value is different
                        // from original value we need to provide original value for this field
    @Input('ax-case-sensitive')
    caseSensitive = false;

    constructor(private elementRef: ElementRef) {
    }

    ngAfterViewInit() {
        this.insertHtmlTagsToHighlightSubstring();
    }

    insertHtmlTagsToHighlightSubstring() {
        // TODO add support for highlighting each occurrence of searched string

        if (this.highlightSubstring && this.highlightSubstring.length) {
            if (typeof this.originalValue === 'string') {
                let newString = this.elementRef.nativeElement.textContent;
                let highlightSubstring = this.highlightSubstring;
                let subStringLength = highlightSubstring.length;
                let substringPosition = this.caseSensitive ?
                        newString.indexOf(highlightSubstring) :
                        newString.toLowerCase().indexOf(highlightSubstring.toLowerCase());
                let endOfSubstringPosition = substringPosition + subStringLength;
                if (substringPosition !== -1) {
                    // add close markup
                    newString = `${newString.slice(0, endOfSubstringPosition)}</div>${newString.slice(endOfSubstringPosition)}`;
                    // add open markup
                    newString =
                        `${newString.slice(0, substringPosition)}<div class="ax-highlight-substring">${newString.slice(substringPosition)}`;

                    this.elementRef.nativeElement.innerHTML = newString;
                }
            } else if (typeof this.originalValue === 'number') {
                // console.log('Number value', this.originalValue)
            }
        }
    }
}

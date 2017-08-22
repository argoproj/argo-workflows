import { Component, OnInit, HostListener } from '@angular/core';
import { ActivatedRoute, Router, UrlSerializer } from '@angular/router';
import { ContentService } from '../../../services';

@Component({
    selector: 'ax-tutorial',
    templateUrl: './tutorial.component.html',
    styles: [ require('./tutorial.component.scss') ],
})
export class TutorialComponent implements OnInit {

    public currentStep = 1;
    public currentStepContent = '';
    public currentStepTitle = '';
    public currentStepIcon = '';

    constructor(
        private contentService: ContentService, private route: ActivatedRoute, private router: Router, private serializer: UrlSerializer) {
    }

    get pointerPosition(): number {
        let nav = this.getCurrentStepNavigation();
        if (nav) {
            return nav.offset().top + nav.height() / 2;
        }
        return 0;
    }

    get isLastStep(): boolean {
        return this.currentStep === this.getNavItems().length;
    }

    get isFirstStep(): boolean {
        return this.currentStep === 1;
    }

    public ngOnInit() {
        setTimeout(() => {
            this.route.queryParams.subscribe(params => {
                if (params['step']) {
                    this.currentStep = parseInt(params['step'], 10);
                } else {
                    let activeNavIndex = this.getNavItems().toArray().findIndex(navItem => $(navItem).parent().hasClass('active'));
                    if (activeNavIndex === -1) {
                        this.currentStep = 1;
                    } else {
                        this.currentStep = activeNavIndex + 1;
                    }
                }
                this.loadCurrentStep();
            });
        });
    }

    public nextStep() {
        this.stepBy(1);
    }

    public prevStep() {
        this.stepBy(-1);
    }

    public close() {
        let url = this.serializer.parse(this.router.url);
        delete url.queryParams['step'];
        delete url.queryParams['tutorial'];
        this.router.navigateByUrl(url.toString());
    }


    @HostListener('window:resize')
    public onWindowResize() {
        // Force recalculating pointerPosition on window resize
    }

    private stepBy(offset: number) {
        let url = this.serializer.parse(this.router.url);
        url.queryParams['step'] = (this.currentStep + offset).toString();
        this.router.navigateByUrl(url.toString());
    }

    private getNavItems() {
        return $('.nav__item_tutorial');
    }

    private getCurrentStepNavigation(): JQuery {
        let navItems = this.getNavItems();
        let itemIndex = Math.min(this.currentStep, navItems.length) - 1;
        return $(navItems[itemIndex]);
    }

    private loadCurrentStep() {
        let nav = this.getCurrentStepNavigation();
        let name = nav.attr('data-tutorial');
        this.currentStepTitle = nav.find('.nav__name').text();
        this.currentStepIcon = nav.find('.nav__ico').find('i').attr('class');
        this.contentService.getTutorial(name).then(content => this.currentStepContent = content).catch(() => this.currentStepContent = '');
    }
}

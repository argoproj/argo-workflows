import { Component, OnInit } from '@angular/core';
import { EventsService } from '../services';

@Component({
    selector: 'ax-help',
    templateUrl: './help.component.html',
    styleUrls: [ './help.component.scss' ],
})
export class HelpComponent implements OnInit {

  private pageTitle = 'Docs';

  constructor(private eventsService: EventsService) {
  }

  public ngOnInit() {
    this.eventsService.setPageTitle.emit(this.pageTitle);
  }
}

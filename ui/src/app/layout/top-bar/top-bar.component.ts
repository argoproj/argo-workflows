import { Component, OnInit } from '@angular/core';
import { EventsService } from '../../services';

@Component({
  selector: 'ax-top-bar',
  templateUrl: './top-bar.html',
  styleUrls: ['./top-bar.component.scss'],
})
export class TopBarComponent implements OnInit {

  public pageTitle = '';

  constructor(private eventsService: EventsService) {
  }

  public ngOnInit() {
    this.eventsService.setPageTitle.subscribe(title => {
      this.pageTitle = title;
    });
  }
}

import { Component } from '@angular/core';

@Component({
  selector: 'ax-top-bar',
  templateUrl: './top-bar.html',
  styleUrls: ['./top-bar.component.scss'],
})
export class TopBarComponent {

  public isGlobalSearchVisible: boolean;
  public pageTitle = 'Timeline';
}

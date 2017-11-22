import { Component } from '@angular/core';

@Component({
  selector: 'ax-layout',
  templateUrl: './layout.html',
  styleUrls: ['./layout.scss'],
})
export class LayoutComponent {
  public hiddenScrollbar: boolean;
  public openedPanelOffCanvas: boolean;
  public openedNav: boolean;

  public toggleNav(status?: boolean) {
    this.openedNav = typeof status !== 'undefined' ? status : !this.openedNav;
  }

}

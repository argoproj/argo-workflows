import { Component, Output, EventEmitter, Input } from '@angular/core';

@Component({
  selector: 'ax-navigation',
  templateUrl: './navigation.html',
  styleUrls: ['./navigation.component.scss'],
})
export class NavigationComponent {

  private version: string;

  constructor() {
  }
}

import { Component, Output, EventEmitter, Input } from '@angular/core';
import { Router } from '@angular/router';

@Component({
  selector: 'ax-navigation',
  templateUrl: './navigation.html',
  styleUrls: ['./navigation.component.scss'],
})
export class NavigationComponent {

  public loading = false;

  @Input()
  public blocked = false;

  @Input()
  public show: boolean;

  @Output()
  public onToggleNav: EventEmitter<any> = new EventEmitter();

  @Output()
  public onCloseNavPanel: EventEmitter<any> = new EventEmitter();

  private version: string;

  constructor(private router: Router) {
  }

  public onClosePanel() {
    this.onCloseNavPanel.emit({});
  }

  public close() {
    this.onToggleNav.emit(false);
  }

  public open() {
    this.onToggleNav.emit(true);
  }
}

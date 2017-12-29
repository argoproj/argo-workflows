import { Component, Input, Output, EventEmitter } from '@angular/core';
import { ToolbarFilters, ToolbarFIltersItem } from './index';

@Component({
  selector: 'ax-toolbar',
  templateUrl: './toolbar.html',
  styleUrls: ['./toolbar.scss'],
})
export class ToolbarComponent {

  @Input()
  public toolbarFilters: ToolbarFilters;

  @Output()
  public onToggleFilter: EventEmitter<string[]> = new EventEmitter();

  public get hasFilters(): boolean {
    return this.toolbarFilters.data.filter(item => {
      return this.toolbarFilters.model.indexOf(item.value) > -1;
    }).length > 0;
  }

  public toggleFilter(option: ToolbarFIltersItem) {
    if (this.toolbarFilters.model.indexOf(option.value) > -1) {
      this.toolbarFilters.model.splice(this.toolbarFilters.model.indexOf(option.value), 1);
    } else {
      this.toolbarFilters.model.push(option.value);
    }

    this.onToggleFilter.next(this.toolbarFilters.model);
  }
}

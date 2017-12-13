export interface ToolbarFIltersItem {
  name: string;
  value: string;
}

export interface ToolbarFilters {
  data: ToolbarFIltersItem[],
  model: string[],
}

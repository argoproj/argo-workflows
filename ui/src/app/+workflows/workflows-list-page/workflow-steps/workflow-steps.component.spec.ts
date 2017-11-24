import { async, ComponentFixture, TestBed } from '@angular/core/testing';

import { WorkflowStepsComponent } from './workflow-steps.component';

describe('WorkflowStepsComponent', () => {
  let component: WorkflowStepsComponent;
  let fixture: ComponentFixture<WorkflowStepsComponent>;

  beforeEach(async(() => {
    TestBed.configureTestingModule({
      declarations: [ WorkflowStepsComponent ]
    })
    .compileComponents();
  }));

  beforeEach(() => {
    fixture = TestBed.createComponent(WorkflowStepsComponent);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });
});

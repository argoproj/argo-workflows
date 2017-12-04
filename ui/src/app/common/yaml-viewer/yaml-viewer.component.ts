import { Component, Input, ElementRef } from '@angular/core';
import * as models from '../../models';
import { DomSanitizer, SafeHtml } from '@angular/platform-browser';
import { ViewUtils } from '../view-utils';
import * as yaml from 'yamljs';
import { WorkflowTree } from '../index';

@Component({
  selector: 'ax-yaml-viewer',
  template: '<div class="yaml-viewer" [innerHTML]="html"></div>',
  styleUrls: [ './yaml-viewer.scss' ],
})
export class YamlViewerComponent {

  public html: SafeHtml;

  constructor(private sanitized: DomSanitizer, private container: ElementRef) {}

  @Input()
  public set input(value: {tree: WorkflowTree, selectedStep: string }) {
    if (value.tree) {
      const yamlString = value.tree.workflow.spec.templates.map(item => {
        let itemStr = yaml.stringify(item, 4, 1);
        if (value.selectedStep) {
          itemStr = this.highlightStep(item, value.selectedStep, itemStr);
        }
        itemStr = this.addCounterToDisplayedFiles(itemStr);
        return itemStr;
      }).join('\n\n');
      this.html = this.sanitized.bypassSecurityTrustHtml(yamlString);
      this.navigateToSelection();
    } else {
      this.html = this.sanitized.bypassSecurityTrustHtml('');
    }
  }

  private highlightStep(template: models.Template, highlightedStepName: string, yamlString: string) {
    let firstLineStepToHighlight = null;
    let lastLineStepToHighlight = null;
    const step = (template.steps || []).reduce((first, second) => first.concat(second), []).find(item => item.name === highlightedStepName);
    if (step) {
      const stepLines = yaml.stringify(step, 1, 1).split('\n');
      firstLineStepToHighlight = `name: ${highlightedStepName}`;
      lastLineStepToHighlight = stepLines[stepLines.length - 2];
    }

    if (firstLineStepToHighlight && lastLineStepToHighlight) {
      let newYamlString = '';
      let isLinePartOfStepToHighlight = false;

      yamlString.split('\n').forEach((line: string, index) => {
        if (line.indexOf(firstLineStepToHighlight) !== -1) {
          isLinePartOfStepToHighlight = true;
        }
        if (isLinePartOfStepToHighlight) {
          newYamlString = `${newYamlString}<span>${line}</span>\n`;
          if (line.indexOf(lastLineStepToHighlight) > -1) {
            isLinePartOfStepToHighlight = false;
          }
        } else {
          newYamlString = `${newYamlString}${line}\n`;
        }
      });

      yamlString = newYamlString;
    }
    return yamlString;
  }

  private navigateToSelection() {
    setTimeout(() => {
      const viewerHighlight = $('li.highlight', this.container.nativeElement).first();
      if (viewerHighlight.length > 0) {
        const parent = $(ViewUtils.scrollParent(viewerHighlight));
        parent.animate({
          scrollTop: viewerHighlight.offset().top + parent.scrollTop() - window.pageYOffset - parent.height() / 2
        });
      }
    });
  }

  private addCounterToDisplayedFiles(multilineString: string): string {
    const newMultilineStringWithCounters: string[] = ['<ol>'];
    multilineString.split('\n').forEach(item => {
      if (item !== '') {
        if (item.indexOf('<span>') !== -1) {
          item = `<li class="highlight">${item}</li>`;
        } else {
          item = `<li>${item}</li>`;
        }
      }
      newMultilineStringWithCounters.push(item);
    });
    newMultilineStringWithCounters.push('</ol>');
    return newMultilineStringWithCounters.join('\n');
  }

  // Removes system information fields which are added in run time
  private removeSystemInfo(item) {
    if (typeof item !== 'object') {
      return item;
    }
    item = Object.assign({}, item, {
      id: undefined,
      revision: undefined,
      service_id: undefined,
      status: undefined,
      cost: undefined,
      create_time: undefined,
      launch_time: undefined,
      end_time: undefined,
      wait_time: undefined,
      run_time: undefined,
      average_runtime: undefined,
      artifact_nums: undefined,
      artifact_size: undefined,
      artifact_tags: undefined,
      jobs_fail: undefined,
      jobs_success: undefined,
      is_success: undefined,
      is_failed: undefined,
    });
    for (const key in item) {
      if (item.hasOwnProperty(key)) {
        if (Array.isArray(item[key])) {
          item[key] = item[key].map(this.removeSystemInfo.bind(this));
        } else if (typeof item[key] === 'object') {
          item[key] = this.removeSystemInfo(item[key]);
        }
      }
    }
    return item;
  }
}

import { NgModule } from '@angular/core';

import { TranslateModule } from 'ng2-translate/ng2-translate';

import { AccountStatePipe } from './accountState.pipe';
import { BytesToGbPipe } from './bytesToGb.pipe';
import { CapitalizePipe } from './capitalize.pipe';
import { CapitalizeSourceNamePipe } from './capitalizeSourceName.pipe';
import { CentsToDollarsPipe } from './centsToDollars.pipe';
import { ContainerRegistryTypePipe } from './containerRegistryType.pipe';
import { DatePipe } from './date.pipe';
import { FilterByValuesInListPipe } from './filterByValuesInList.pipe';
import { FloatToPercentsPipe } from './floatToPercents.pipe';
import { FullTimePipe } from './fullTime.pipe';
import { HumanizeTimePipe } from './humanizeTime.pipe';
import { MbToGbPipe } from './mbToGb.pipe';
import { RepoNamePipe } from './repoName.pipe';
import { SecondsToMillisecondsPipe } from './secondsToMilliseconds.pipe';
import { ShortDateTimePipe } from './shortDateTime.pipe';
import { ShortRevisionPipe } from './shortRevision.pipe';
import { ShortTimePipe } from './shortTime.pipe';
import { StatusPipe } from './status.pipe';
import { StatusToNumberPipe } from './statusToNumber.pipe';
import { TimestampPipe } from './timestamp.pipe';
import { ToServiceStatusPipe } from './toServiceStatus.pipe';
import { TruncateToPipe } from './truncateTo.pipe';
import { JobStatusPipe } from './jobStatus.pipe';
import { JobTypePipe } from './jobType.pipe';
import { BranchesSortPipe } from './branches-sort.pipe';
import { DurationPipe } from './duration.pipe';
import { BranchesSearchPipe } from './branches-search.pipe';
import { MillisecondsToSecondsPipe } from './millisecondsToSeconds.pipe';
import { LabelsSearchPipe } from './labelsSearch.pipe';
import { TagsSearchPipe } from './tagsSearch.pipe';
import { DeletedStatusPipe } from './deletedStatus.pipe';
import { FilterByPipe } from './filterBy.pipe';
import { KeysPipe } from './keys.pipe';
import { SortFilterMenuPipe } from './sortFilterMenu.pipe';

let pipes: any[] = [
    AccountStatePipe,
    BytesToGbPipe,
    CapitalizePipe,
    CapitalizeSourceNamePipe,
    CentsToDollarsPipe,
    ContainerRegistryTypePipe,
    DatePipe,
    FilterByPipe,
    FilterByValuesInListPipe,
    FloatToPercentsPipe,
    FullTimePipe,
    HumanizeTimePipe,
    MbToGbPipe,
    RepoNamePipe,
    SecondsToMillisecondsPipe,
    ShortDateTimePipe,
    ShortRevisionPipe,
    ShortTimePipe,
    StatusPipe,
    StatusToNumberPipe,
    TimestampPipe,
    ToServiceStatusPipe,
    TruncateToPipe,
    JobStatusPipe,
    JobTypePipe,
    BranchesSortPipe,
    DurationPipe,
    BranchesSearchPipe,
    MillisecondsToSecondsPipe,
    LabelsSearchPipe,
    TagsSearchPipe,
    DeletedStatusPipe,
    KeysPipe,
    SortFilterMenuPipe,
];

@NgModule({
    declarations: pipes,
    exports: pipes.concat(TranslateModule)
})
export class PipesModule {
}

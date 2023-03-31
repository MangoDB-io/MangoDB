// Code generated by "stringer -linecomment -type ErrorCode"; DO NOT EDIT.

package commonerrors

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[errUnset-0]
	_ = x[errInternalError-1]
	_ = x[ErrBadValue-2]
	_ = x[ErrFailedToParse-9]
	_ = x[ErrTypeMismatch-14]
	_ = x[ErrNamespaceNotFound-26]
	_ = x[ErrUnsuitableValueType-28]
	_ = x[ErrConflictingUpdateOperators-40]
	_ = x[ErrCursorNotFound-43]
	_ = x[ErrNamespaceExists-48]
	_ = x[ErrInvalidID-53]
	_ = x[ErrEmptyName-56]
	_ = x[ErrCommandNotFound-59]
	_ = x[ErrCannotCreateIndex-67]
	_ = x[ErrInvalidNamespace-73]
	_ = x[ErrIndexOptionsConflict-85]
	_ = x[ErrIndexKeySpecsConflict-86]
	_ = x[ErrOperationFailed-96]
	_ = x[ErrDocumentValidationFailure-121]
	_ = x[ErrNotImplemented-238]
	_ = x[ErrMechanismUnavailable-334]
	_ = x[ErrDuplicateKey-11000]
	_ = x[ErrStageGroupInvalidFields-15947]
	_ = x[ErrStageGroupID-15948]
	_ = x[ErrStageGroupMissingID-15955]
	_ = x[ErrMatchBadExpression-15959]
	_ = x[ErrSortBadExpression-15973]
	_ = x[ErrSortBadValue-15974]
	_ = x[ErrSortBadOrder-15975]
	_ = x[ErrSortMissingKey-15976]
	_ = x[ErrStageUnwindWrongType-15981]
	_ = x[ErrPathContainsEmptyElement-15998]
	_ = x[ErrFieldPathInvalidName-16410]
	_ = x[ErrGroupInvalidFieldPath-16872]
	_ = x[ErrGroupUndefinedVariable-17276]
	_ = x[ErrInvalidArg-28667]
	_ = x[ErrSliceFirstArg-28724]
	_ = x[ErrStageUnwindNoPath-28812]
	_ = x[ErrStageUnwindNoPrefix-28818]
	_ = x[ErrProjectionInEx-31253]
	_ = x[ErrProjectionExIn-31254]
	_ = x[ErrStageCountNonString-40156]
	_ = x[ErrStageCountNonEmptyString-40157]
	_ = x[ErrStageCountBadPrefix-40158]
	_ = x[ErrStageCountBadValue-40160]
	_ = x[ErrStageGroupUnaryOperator-40237]
	_ = x[ErrStageGroupMultipleAccumulator-40238]
	_ = x[ErrStageInvalid-40323]
	_ = x[ErrStageGroupInvalidAccumulator-40234]
	_ = x[ErrEmptyFieldPath-40352]
	_ = x[ErrMissingField-40414]
	_ = x[ErrFailedToParseInput-40415]
	_ = x[ErrFreeMonitoringDisabled-50840]
	_ = x[ErrValueNegative-51024]
	_ = x[ErrRegexOptions-51075]
	_ = x[ErrRegexMissingParen-51091]
	_ = x[ErrBadRegexOption-51108]
	_ = x[ErrDuplicateField-4822819]
}

const _ErrorCode_name = "UnsetInternalErrorBadValueFailedToParseTypeMismatchNamespaceNotFoundUnsuitableValueTypeConflictingUpdateOperatorsCursorNotFoundNamespaceExistsInvalidIDEmptyNameCommandNotFoundCannotCreateIndexInvalidNamespaceIndexOptionsConflictIndexKeySpecsConflictOperationFailedDocumentValidationFailureNotImplementedMechanismUnavailableLocation11000Location15947Location15948Location15955Location15959Location15973Location15974Location15975Location15976Location15981Location15998Location16410Location16872Location17276Location28667Location28724Location28812Location28818Location31253Location31254Location40156Location40157Location40158Location40160Location40234Location40237Location40238Location40323Location40352Location40414Location40415Location50840Location51024Location51075Location51091Location51108Location4822819"

var _ErrorCode_map = map[ErrorCode]string{
	0:       _ErrorCode_name[0:5],
	1:       _ErrorCode_name[5:18],
	2:       _ErrorCode_name[18:26],
	9:       _ErrorCode_name[26:39],
	14:      _ErrorCode_name[39:51],
	26:      _ErrorCode_name[51:68],
	28:      _ErrorCode_name[68:87],
	40:      _ErrorCode_name[87:113],
	43:      _ErrorCode_name[113:127],
	48:      _ErrorCode_name[127:142],
	53:      _ErrorCode_name[142:151],
	56:      _ErrorCode_name[151:160],
	59:      _ErrorCode_name[160:175],
	67:      _ErrorCode_name[175:192],
	73:      _ErrorCode_name[192:208],
	85:      _ErrorCode_name[208:228],
	86:      _ErrorCode_name[228:249],
	96:      _ErrorCode_name[249:264],
	121:     _ErrorCode_name[264:289],
	238:     _ErrorCode_name[289:303],
	334:     _ErrorCode_name[303:323],
	11000:   _ErrorCode_name[323:336],
	15947:   _ErrorCode_name[336:349],
	15948:   _ErrorCode_name[349:362],
	15955:   _ErrorCode_name[362:375],
	15959:   _ErrorCode_name[375:388],
	15973:   _ErrorCode_name[388:401],
	15974:   _ErrorCode_name[401:414],
	15975:   _ErrorCode_name[414:427],
	15976:   _ErrorCode_name[427:440],
	15981:   _ErrorCode_name[440:453],
	15998:   _ErrorCode_name[453:466],
	16410:   _ErrorCode_name[466:479],
	16872:   _ErrorCode_name[479:492],
	17276:   _ErrorCode_name[492:505],
	28667:   _ErrorCode_name[505:518],
	28724:   _ErrorCode_name[518:531],
	28812:   _ErrorCode_name[531:544],
	28818:   _ErrorCode_name[544:557],
	31253:   _ErrorCode_name[557:570],
	31254:   _ErrorCode_name[570:583],
	40156:   _ErrorCode_name[583:596],
	40157:   _ErrorCode_name[596:609],
	40158:   _ErrorCode_name[609:622],
	40160:   _ErrorCode_name[622:635],
	40234:   _ErrorCode_name[635:648],
	40237:   _ErrorCode_name[648:661],
	40238:   _ErrorCode_name[661:674],
	40323:   _ErrorCode_name[674:687],
	40352:   _ErrorCode_name[687:700],
	40414:   _ErrorCode_name[700:713],
	40415:   _ErrorCode_name[713:726],
	50840:   _ErrorCode_name[726:739],
	51024:   _ErrorCode_name[739:752],
	51075:   _ErrorCode_name[752:765],
	51091:   _ErrorCode_name[765:778],
	51108:   _ErrorCode_name[778:791],
	4822819: _ErrorCode_name[791:806],
}

func (i ErrorCode) String() string {
	if str, ok := _ErrorCode_map[i]; ok {
		return str
	}
	return "ErrorCode(" + strconv.FormatInt(int64(i), 10) + ")"
}

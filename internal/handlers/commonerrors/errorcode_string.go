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
	_ = x[ErrIndexNotFound-27]
	_ = x[ErrUnsuitableValueType-28]
	_ = x[ErrConflictingUpdateOperators-40]
	_ = x[ErrCursorNotFound-43]
	_ = x[ErrNamespaceExists-48]
	_ = x[ErrInvalidID-53]
	_ = x[ErrEmptyName-56]
	_ = x[ErrCommandNotFound-59]
	_ = x[ErrInvalidOptions-72]
	_ = x[ErrInvalidNamespace-73]
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
	_ = x[ErrPathContainsEmptyElement-15998]
	_ = x[ErrGroupInvalidFieldPath-16872]
	_ = x[ErrGroupUndefinedVariable-17276]
	_ = x[ErrInvalidArg-28667]
	_ = x[ErrSliceFirstArg-28724]
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

const _ErrorCode_name = "UnsetInternalErrorBadValueFailedToParseTypeMismatchNamespaceNotFoundIndexNotFoundUnsuitableValueTypeConflictingUpdateOperatorsCursorNotFoundNamespaceExistsInvalidIDEmptyNameCommandNotFoundInvalidOptionsInvalidNamespaceOperationFailedDocumentValidationFailureNotImplementedMechanismUnavailableLocation11000Location15947Location15948Location15955Location15959Location15973Location15974Location15975Location15976Location15998Location16872Location17276Location28667Location28724Location31253Location31254Location40156Location40157Location40158Location40160Location40234Location40237Location40238Location40323Location40352Location40414Location40415Location50840Location51024Location51075Location51091Location51108Location4822819"

var _ErrorCode_map = map[ErrorCode]string{
	0:       _ErrorCode_name[0:5],
	1:       _ErrorCode_name[5:18],
	2:       _ErrorCode_name[18:26],
	9:       _ErrorCode_name[26:39],
	14:      _ErrorCode_name[39:51],
	26:      _ErrorCode_name[51:68],
	27:      _ErrorCode_name[68:81],
	28:      _ErrorCode_name[81:100],
	40:      _ErrorCode_name[100:126],
	43:      _ErrorCode_name[126:140],
	48:      _ErrorCode_name[140:155],
	53:      _ErrorCode_name[155:164],
	56:      _ErrorCode_name[164:173],
	59:      _ErrorCode_name[173:188],
	72:      _ErrorCode_name[188:202],
	73:      _ErrorCode_name[202:218],
	96:      _ErrorCode_name[218:233],
	121:     _ErrorCode_name[233:258],
	238:     _ErrorCode_name[258:272],
	334:     _ErrorCode_name[272:292],
	11000:   _ErrorCode_name[292:305],
	15947:   _ErrorCode_name[305:318],
	15948:   _ErrorCode_name[318:331],
	15955:   _ErrorCode_name[331:344],
	15959:   _ErrorCode_name[344:357],
	15973:   _ErrorCode_name[357:370],
	15974:   _ErrorCode_name[370:383],
	15975:   _ErrorCode_name[383:396],
	15976:   _ErrorCode_name[396:409],
	15998:   _ErrorCode_name[409:422],
	16872:   _ErrorCode_name[422:435],
	17276:   _ErrorCode_name[435:448],
	28667:   _ErrorCode_name[448:461],
	28724:   _ErrorCode_name[461:474],
	31253:   _ErrorCode_name[474:487],
	31254:   _ErrorCode_name[487:500],
	40156:   _ErrorCode_name[500:513],
	40157:   _ErrorCode_name[513:526],
	40158:   _ErrorCode_name[526:539],
	40160:   _ErrorCode_name[539:552],
	40234:   _ErrorCode_name[552:565],
	40237:   _ErrorCode_name[565:578],
	40238:   _ErrorCode_name[578:591],
	40323:   _ErrorCode_name[591:604],
	40352:   _ErrorCode_name[604:617],
	40414:   _ErrorCode_name[617:630],
	40415:   _ErrorCode_name[630:643],
	50840:   _ErrorCode_name[643:656],
	51024:   _ErrorCode_name[656:669],
	51075:   _ErrorCode_name[669:682],
	51091:   _ErrorCode_name[682:695],
	51108:   _ErrorCode_name[695:708],
	4822819: _ErrorCode_name[708:723],
}

func (i ErrorCode) String() string {
	if str, ok := _ErrorCode_map[i]; ok {
		return str
	}
	return "ErrorCode(" + strconv.FormatInt(int64(i), 10) + ")"
}

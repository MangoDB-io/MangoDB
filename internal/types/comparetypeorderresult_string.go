// Code generated by "stringer -linecomment -type compareTypeOrderResult"; DO NOT EDIT.

package types

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[nullDataType-1]
	_ = x[nanDataType-2]
	_ = x[numbersDataType-3]
	_ = x[stringDataType-4]
	_ = x[documentDataType-5]
	_ = x[arrayDataType-6]
	_ = x[binDataType-7]
	_ = x[objectIdDataType-8]
	_ = x[booleanDataType-9]
	_ = x[dateDataType-10]
	_ = x[timestampDataType-11]
	_ = x[regexDataType-12]
}

const _compareTypeOrderResult_name = "nullDataTypenanDataTypenumbersDataTypestringDataTypeTODO: https://github.com/FerretDB/FerretDB/issues/457TODO: https://github.com/FerretDB/FerretDB/issues/457binDataTypeobjectIdDataTypebooleanDataTypedateDataTypetimestampDataTyperegexDataType"

var _compareTypeOrderResult_index = [...]uint8{0, 12, 23, 38, 52, 105, 158, 169, 185, 200, 212, 229, 242}

func (i compareTypeOrderResult) String() string {
	i -= 1
	if i >= compareTypeOrderResult(len(_compareTypeOrderResult_index)-1) {
		return "compareTypeOrderResult(" + strconv.FormatInt(int64(i+1), 10) + ")"
	}
	return _compareTypeOrderResult_name[_compareTypeOrderResult_index[i]:_compareTypeOrderResult_index[i+1]]
}

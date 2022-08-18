// Copyright 2021 FerretDB Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tigris

import (
	"encoding/json"
	"github.com/FerretDB/FerretDB/internal/handlers/common"
	"github.com/FerretDB/FerretDB/internal/types"
)

// getJSONSchema returns a masrshaled JSON schema received from validator -> $jsonSchema.
func getJSONSchema(doc *types.Document) ([]byte, error) {
	v, err := doc.Get("validator")
	if err != nil {
		return nil, common.NewErrorMsg(common.ErrBadValue, "required parameter `validator` is missing")
	}

	schema, err := v.(*types.Document).Get("$jsonSchema")
	if err != nil {
		return nil, common.NewErrorMsg(common.ErrBadValue, "required parameter `$jsonSchema` is missing")
	}

	return json.Marshal(schema.(*types.Document))
}

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

package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"

	"github.com/FerretDB/FerretDB/internal/clientconn/conninfo"
	"github.com/FerretDB/FerretDB/internal/handler/common"
	"github.com/FerretDB/FerretDB/internal/handler/handlererrors"
	"github.com/FerretDB/FerretDB/internal/types"
	"github.com/FerretDB/FerretDB/internal/util/lazyerrors"
	"github.com/FerretDB/FerretDB/internal/util/must"
	"github.com/FerretDB/FerretDB/internal/wire"
	"github.com/xdg-go/scram"
	"go.uber.org/zap"
)

// MsgSASLStart implements `saslStart` command.
func (h *Handler) MsgSASLStart(ctx context.Context, msg *wire.OpMsg) (*wire.OpMsg, error) {
	document, err := msg.Document()
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	dbName, err := common.GetRequiredParam[string](document, "$db")
	if err != nil {
		return nil, err
	}

	// TODO https://github.com/FerretDB/FerretDB/issues/3008

	// database name typically is either "$external" or "admin"
	// we can't use it to query the database
	_ = dbName

	mechanism, err := common.GetRequiredParam[string](document, "mechanism")
	if err != nil {
		return nil, lazyerrors.Error(err)
	}

	var username, password string

	var response string

	var sconv *types.ScramConv

	plain := true

	switch mechanism {
	case "PLAIN":
		username, password, err = saslStartPlain(document)
		if err != nil {
			return nil, err
		}

		conninfo.Get(ctx).SetAuth(username, password)

	case "SCRAM-SHA-256":
		response, sconv, err = saslStartSCRAM(document)
		if err != nil {
			return nil, err
		}

		plain = false

		h.L.Debug(
			"saslStart",
			zap.String("response", response),
			zap.String("user", sconv.Conv.Username()),
			zap.Bool("authenticated", sconv.Conv.Valid()),
		)

	default:
		msg := fmt.Sprintf("Unsupported authentication mechanism %q.\n", mechanism) +
			"See https://docs.ferretdb.io/security/authentication/ for more details."
		return nil, handlererrors.NewCommandErrorMsgWithArgument(handlererrors.ErrAuthenticationFailed, msg, "mechanism")
	}

	var emptyPayload types.Binary
	var reply wire.OpMsg
	d := must.NotFail(types.NewDocument(
		"conversationId", int32(1),
		"done", false,
		"payload", emptyPayload,
		"ok", float64(1),
	))

	// TODO confirm if this is even needed or if speculativeAuthenticate is always used and is sent in an OP_QUERY
	if !plain {
		// remove top-level fields
		d.Remove("conversationId")
		d.Remove("done")
		d.Remove("payload")

		// create a speculative conversation document for SCRAM authentication
		d.Set("speculativeAuthenticate", must.NotFail(
			types.NewDocument(
				"conversationId", int32(1),
				"done", false,
				"payload", response,
				"ok", float64(1),
			),
		))
	}

	must.NoError(reply.SetSections(wire.OpMsgSection{
		Documents: []*types.Document{d},
	}))

	return &reply, nil
}

// saslStartPlain extracts username and password from PLAIN `saslStart` payload.
func saslStartPlain(doc *types.Document) (string, string, error) {
	var payload []byte

	// some drivers send payload as a string
	stringPayload, err := common.GetRequiredParam[string](doc, "payload")
	if err == nil {
		if payload, err = base64.StdEncoding.DecodeString(stringPayload); err != nil {
			return "", "", handlererrors.NewCommandErrorMsgWithArgument(
				handlererrors.ErrBadValue,
				fmt.Sprintf("Invalid payload: %v", err),
				"payload",
			)
		}
	}

	// most drivers follow spec and send payload as a binary
	binaryPayload, err := common.GetRequiredParam[types.Binary](doc, "payload")
	if err == nil {
		payload = binaryPayload.B
	}

	// as spec's payload should be binary, we return an error mentioned binary as expected type
	if payload == nil {
		return "", "", err
	}

	fields := bytes.Split(payload, []byte{0})
	if l := len(fields); l != 3 {
		return "", "", handlererrors.NewCommandErrorMsgWithArgument(
			handlererrors.ErrTypeMismatch,
			fmt.Sprintf("Invalid payload: expected 3 fields, got %d", l),
			"payload",
		)
	}

	authzid, authcid, passwd := fields[0], fields[1], fields[2]

	// Some drivers (Go) send empty authorization identity (authzid),
	// while others (Java) set it to the same value as authentication identity (authcid)
	// (see https://www.rfc-editor.org/rfc/rfc4616.html).
	// Ignore authzid for now.
	_ = authzid

	return string(authcid), string(passwd), nil
}

func saslStartSCRAM(doc *types.Document) (string, *types.ScramConv, error) {
	// TODO store the SCRAM-SHA-1 and SCRAM-SHA-256 credentials in the 'admin.system.users' namespace
	// when a user is initially created
	// credentials: {
	//   'PLAIN': {
	//     algo: 'argon2id',
	//     t: int32(3),
	//     p: int32(4),
	//     m: int32(65536),
	//     hash: types.Binary{…},
	//     salt: types.Binary{…},
	//   },
	//   'SCRAM-SHA-1': {
	//     iterationCount: 10000,
	//     salt: 'HABkWd1LV6tLbXNsqrXc5w==',
	//     storedKey: 'BkeR3SlFOm3xxER4qtGDgFu4imw=',
	//     serverKey: 'WNDA6r92qAKZvMr0J6mbxRJGuQo='
	//   },
	//   'SCRAM-SHA-256': {
	//     iterationCount: 15000,
	//     salt: '7jW5ZOczj05P4wyNc21OikIuSliPN9rw4sEoGQ==',
	//     storedKey: 'F8hTLrnZscuuszfrh+4nupyjPA40cp+gfzy1Hsc3O3c=',
	//     serverKey: 'd4P+d81D31XHwvfQA3jwgTmkivZfXTD/nBASm77Dwv0='
	//   }
	// }

	var payload []byte

	binaryPayload, err := common.GetRequiredParam[types.Binary](doc, "payload")
	if err == nil {
		payload = binaryPayload.B
	}

	var (
		salt      = []byte{238, 53, 185, 100, 231, 51, 143, 78, 79, 227, 12, 141, 115, 109, 78, 138, 66, 46, 74, 88, 143, 55, 218, 240, 226, 193, 40, 25}
		storedKey = []byte{23, 200, 83, 46, 185, 217, 177, 203, 174, 179, 55, 235, 135, 238, 39, 186, 156, 163, 60, 14, 52, 114, 159, 160, 127, 60, 181, 30, 199, 55, 59, 119}
		serverKey = []byte{119, 131, 254, 119, 205, 67, 223, 85, 199, 194, 247, 208, 3, 120, 240, 129, 57, 164, 138, 246, 95, 93, 48, 255, 156, 16, 18, 155, 190, 195, 194, 253}
	)

	var response string

	cl := scram.CredentialLookup(func(s string) (scram.StoredCredentials, error) {
		kf := scram.KeyFactors{
			Salt:  string(salt),
			Iters: 15000,
		}

		return scram.StoredCredentials{
			KeyFactors: kf,
			StoredKey:  storedKey,
			ServerKey:  serverKey,
		}, nil
	})

	ss, err := scram.SHA256.NewServer(cl)
	must.NoError(err)

	conv := ss.NewConversation()
	response, err = conv.Step(string(payload))
	must.NoError(err)

	sconv := &types.ScramConv{
		Salt:      salt,
		StoredKey: storedKey,
		ServerKey: serverKey,
		Conv:      conv,
	}

	return response, sconv, nil
}

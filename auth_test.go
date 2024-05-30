// Copyright © 2015-2023 Brett Vickers.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ntp

import (
	"bytes"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOnlineAuthenticatedQuery(t *testing.T) {
	// By default, this unit test is skipped, because it requires a local NTP
	// server to be running and configured with known symmetric authentication
	// keys.
	//
	// To run this test, you must execute it with "-args test_auth". For
	// example:
	//
	//    go test -v -run TestOnlineAuthenticatedQuery -args test_auth
	//
	// You must also run a local NTP server configured with the following
	// trusted symmetric keys (shown in chrony.keys format):
	//
	// 1  MD5        ASCII:cvuZyN4C8HX8hNcAWDWp
	// 2  SHA1       HEX:6931564b4a5a5045766c55356b30656c7666316c
	// 3  SHA256     HEX:7133736e777057764256777739706a5533326164
	// 4  SHA512     HEX:597675555446585868494d447543425971526e74
	// 5  AES128     HEX:68663033736f77706568707164304049
	// 6  AES256     HEX:47cb76a9a507cf26dc00eb0935f082f390f10308c3e0d58716273a63259a758a

	skip := true
	for _, arg := range os.Args {
		if arg == "test_auth" {
			skip = false
		}
	}
	if skip {
		t.Skip("Skipping authentication tests. Enable with -args test_auth")
		return
	}

	errAuthFail := errors.New("timeout")

	cases := []struct {
		Type        AuthType
		Key         string
		KeyID       uint16
		ExpectedErr error
	}{
		// KeyID 1 (MD5)
		{AuthMD5, "cvuZyN4C8HX8hNcAWDWp", 1, nil},
		{AuthMD5, "ASCII:cvuZyN4C8HX8hNcAWDWp", 1, nil},
		{AuthMD5, "6376755a794e344338485838684e634157445770", 1, nil},
		{AuthMD5, "HEX:6376755a794e344338485838684e634157445770", 1, nil},
		{AuthMD5, "", 1, ErrInvalidAuthKey},
		{AuthMD5, "HEX:6376755a794e344338485838684e63415744577", 1, ErrInvalidAuthKey},
		{AuthMD5, "HEX:6376755a794e344338485838684e63415744577g", 1, ErrInvalidAuthKey},
		{AuthMD5, "ASCII:XvuZyN4C8HX8hNcAWDWp", 1, errAuthFail},
		{AuthMD5, "ASCII:cvuZyN4C8HX8hNcAWDWp", 2, errAuthFail},
		{AuthSHA1, "ASCII:cvuZyN4C8HX8hNcAWDWp", 1, errAuthFail},

		// KeyID 2 (SHA1)
		{AuthSHA1, "HEX:6931564b4a5a5045766c55356b30656c7666316c", 2, nil},
		{AuthSHA1, "HEX:6931564b4a5a5045766c55356b30656c7666316c", 2, nil},
		{AuthSHA1, "ASCII:i1VKJZPEvlU5k0elvf1l", 2, nil},
		{AuthSHA1, "ASCII:i1VKJZPEvlU5k0elvf1l", 2, nil},
		{AuthSHA1, "", 2, ErrInvalidAuthKey},
		{AuthSHA1, "HEX:0031564b4a5a5045766c55356b30656c7666316c", 2, errAuthFail},
		{AuthSHA1, "HEX:6931564b4a5a5045766c55356b30656c7666316c", 1, errAuthFail},
		{AuthMD5, "HEX:6931564b4a5a5045766c55356b30656c7666316c", 2, errAuthFail},

		// KeyID 3 (SHA256)
		{AuthSHA256, "HEX:7133736e777057764256777739706a5533326164", 3, nil},
		{AuthSHA256, "ASCII:q3snwpWvBVww9pjU32ad", 3, nil},
		{AuthSHA256, "", 3, ErrInvalidAuthKey},
		{AuthSHA256, "HEX:0033736e777057764256777739706a5533326164", 3, errAuthFail},
		{AuthSHA256, "HEX:7133736e777057764256777739706a5533326164", 2, errAuthFail},
		{AuthSHA1, "HEX:7133736e777057764256777739706a5533326164", 3, errAuthFail},

		// // KeyID 4 (SHA512)
		{AuthSHA512, "HEX:597675555446585868494d447543425971526e74", 4, nil},
		{AuthSHA512, "ASCII:YvuUTFXXhIMDuCBYqRnt", 4, nil},
		{AuthSHA512, "", 4, ErrInvalidAuthKey},
		{AuthSHA512, "HEX:007675555446585868494d447543425971526e74", 4, errAuthFail},
		{AuthSHA512, "HEX:597675555446585868494d447543425971526e74", 3, errAuthFail},
		{AuthSHA256, "HEX:597675555446585868494d447543425971526e74", 4, errAuthFail},

		// KeyID 5 (AES128)
		{AuthAES128, "HEX:68663033736f77706568707164304049", 5, nil},
		{AuthAES128, "HEX:68663033736f77706568707164304049fefefefe", 5, nil},
		{AuthAES128, "ASCII:hf03sowpehpqd0@I", 5, nil},
		{AuthAES128, "", 5, ErrInvalidAuthKey},
		{AuthAES128, "HEX:00663033736f77706568707164304049", 5, errAuthFail},
		{AuthAES128, "HEX:68663033736f77706568707164304049", 4, errAuthFail},
		{AuthMD5, "HEX:68663033736f77706568707164304049", 5, errAuthFail},

		// KeyID 6 (AES256)
		{AuthAES256, "HEX:47cb76a9a507cf26dc00eb0935f082f390f10308c3e0d58716273a63259a758a", 6, nil},
		{AuthAES256, "", 6, ErrInvalidAuthKey},
		{AuthAES256, "HEX:00cb76a9a507cf26dc00eb0935f082f390f10308c3e0d58716273a63259a758a", 6, errAuthFail},
		{AuthAES256, "HEX:47cb76a9a507cf26dc00eb0935f082f390f10308c3e0d58716273a63259a758a", 5, errAuthFail},
		{AuthMD5, "HEX:47cb76a9a507cf26dc00eb0935f082f390f10308c3e0d58716273a63259a758a", 6, errAuthFail},
	}

	for i, c := range cases {
		opt := QueryOptions{
			Timeout: 250 * time.Millisecond,
			Auth:    AuthOptions{c.Type, c.Key, c.KeyID},
		}
		r, err := QueryWithOptions(host, opt)
		if c.ExpectedErr == errAuthFail {
			// With old NTP servers, failed authentication leads to Crypto-NAK
			// (ErrAuthFailed). With modern NTP servers, it leads to an I/O
			// timeout error.
			if err != ErrAuthFailed && !strings.Contains(err.Error(), "timeout") {
				t.Errorf("case %d: expected error [%v], got error [%v]\n", i, c.ExpectedErr, err)
			}
			continue
		}
		if c.ExpectedErr != nil && c.ExpectedErr == err {
			continue
		}
		if err == nil {
			err = r.Validate()
			if err != c.ExpectedErr {
				t.Errorf("case %d: expected error [%v], got error [%v]\n", i, c.ExpectedErr, err)
			}
		}
	}
}

func TestOfflineAesCmac(t *testing.T) {
	// Test cases taken from NIST document:
	// https://csrc.nist.gov/CSRC/media/Projects/Cryptographic-Standards-and-Guidelines/documents/examples/AES_CMAC.pdf
	const (
		Key128 = "2b7e1516 28aed2a6 abf71588 09cf4f3c"
		Key192 = "8e73b0f7 da0e6452 c810f32b 809079e5 62f8ead2 522c6b7b"
		Key256 = "603deb10 15ca71be 2b73aef0 857d7781 1f352c07 3b6108d7 2d9810a3 0914dff4"
	)

	const (
		Msg1 = ""
		Msg2 = "6bc1bee2 2e409f96 e93d7e11 7393172a"
		Msg3 = "6bc1bee2 2e409f96 e93d7e11 7393172a ae2d8a57"
		Msg4 = "6bc1bee2 2e409f96 e93d7e11 7393172a ae2d8a57 1e03ac9c 9eb76fac 45af8e51" +
			"30c81c46 a35ce411 e5fbc119 1a0a52ef f69f2445 df4f9b17 ad2b417b e66c3710"
	)

	cases := []struct {
		key       string
		plaintext string
		cmac      string
	}{
		// 128-bit key
		{Key128, Msg1, "bb1d6929 e9593728 7fa37d12 9b756746"},
		{Key128, Msg2, "070a16b4 6b4d4144 f79bdd9d d04a287c"},
		{Key128, Msg3, "7d85449e a6ea19c8 23a7bf78 837dfade"},
		{Key128, Msg4, "51f0bebf 7e3b9d92 fc497417 79363cfe"},

		// 192-bit key
		{Key192, Msg1, "d17ddf46 adaacde5 31cac483 de7a9367"},
		{Key192, Msg2, "9e99a7bf 31e71090 0662f65e 617c5184"},
		{Key192, Msg3, "3d75c194 ed960704 44a9fa7e c740ecf8"},
		{Key192, Msg4, "a1d5df0e ed790f79 4d775896 59f39a11"},

		// 256-bit key
		{Key256, Msg1, "028962f6 1b7bf89e fc6b551f 4667d983"},
		{Key256, Msg2, "28a7023f 452e8f82 bd4bf28d 8c37c35c"},
		{Key256, Msg3, "156727dc 0878944a 023c1fe0 3bad6d93"},
		{Key256, Msg4, "e1992190 549f6ed5 696a2c05 6c315410"},
	}

	for i, c := range cases {
		_ = i
		key, pt, cmac := hexDecode(c.key), hexDecode(c.plaintext), hexDecode(c.cmac)
		result := calcCMAC_AES(pt, key)
		if !bytes.Equal(cmac, result) {
			t.Errorf("case %d: CMACs do not match.\n", i)
		}
	}
}

func hexDecode(s string) []byte {
	s = strings.ReplaceAll(s, " ", "")
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

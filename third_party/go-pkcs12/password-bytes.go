// Copyright 2015, 2018, 2019 Opsmate, Inc. All rights reserved.
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file is a local nanci addition; it does not exist upstream.
// See README.nanci.md.

package pkcs12

import (
	"crypto/x509"
	"errors"
	"unicode/utf16"
	"unicode/utf8"
)

// DecodeChainBytes is [DecodeChain] with the password given as a byte slice
// instead of a string, so the caller can overwrite the password after use.
// Go strings are immutable and cannot be zeroed, so any caller that keeps the
// password in memory must avoid converting it to a string. The password bytes
// are interpreted as UTF-8, exactly as the string form is.
//
// The caller owns password and is responsible for zeroing it; this function
// does not modify it. The intermediate BMP-string encoding allocated here is
// zeroed before returning.
func DecodeChainBytes(pfxData, password []byte) (privateKey interface{}, certificate *x509.Certificate, caCerts []*x509.Certificate, err error) {
	encodedPassword, err := bmpStringBytesZeroTerminated(password)
	if err != nil {
		return nil, nil, nil, err
	}
	defer zeroBytes(encodedPassword)

	return decodeChain(pfxData, encodedPassword)
}

// bmpStringBytesZeroTerminated returns password encoded in UCS-2 with a zero
// terminator. It is [bmpStringZeroTerminated] for a byte-slice password.
func bmpStringBytesZeroTerminated(password []byte) ([]byte, error) {
	ret, err := bmpStringBytes(password)
	if err != nil {
		return nil, err
	}

	return append(ret, 0, 0), nil
}

// bmpStringBytes returns password encoded in UCS-2. It is [bmpString] for a
// byte-slice password: ranging over a string decodes UTF-8 into runes, so this
// decodes the byte slice the same way, invalid bytes included (they become
// utf8.RuneError, which is what ranging over a string yields as well).
func bmpStringBytes(password []byte) ([]byte, error) {
	ret := make([]byte, 0, 2*len(password)+2)

	for i := 0; i < len(password); {
		r, size := utf8.DecodeRune(password[i:])
		i += size
		if t, _ := utf16.EncodeRune(r); t != 0xfffd {
			zeroBytes(ret)
			return nil, errors.New("pkcs12: string contains characters that cannot be encoded in UCS-2")
		}
		ret = append(ret, byte(r/256), byte(r%256))
	}

	return ret, nil
}

// zeroBytes overwrites b with zeros.
func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

// Copyright (c) 2020-2021 Sippy Software, Inc. All rights reserved.
//
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without modification,
// are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
// list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
// this list of conditions and the following disclaimer in the documentation and/or
// other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
// ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
// ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
package sippy_sshaken

import (
	"errors"
	"time"
)

type sshaken_passport struct {
	ppt_hdr_param string
	alg_hdr_param string
	signature     []byte
	Header        sshaken_header
	Payload       sshaken_payload
}

type sshaken_header struct {
	Alg string `json:"alg"`
	Ppt string `json:"ppt"`
	Typ string `json:"typ"`
	X5u string `json:"x5u"`
}

type sshaken_payload struct {
	Attest string       `json:"attest"`
	Dest   sshaken_dest `json:"dest"`
	Iat    int64        `json:"iat"`
	Orig   sshaken_orig `json:"orig"`
	Origid string       `json:"origid"`
}

type sshaken_dest struct {
	TN []string `json:"tn"`
}

type sshaken_orig struct {
	TN string `json:"tn"`
}

func (s *sshaken_passport) Origid() string {
	return s.Payload.Origid
}

func (s *sshaken_passport) Attest() string {
	return s.Payload.Attest
}

func (s *sshaken_passport) X5u() string {
	return s.Header.X5u
}

func (s *sshaken_passport) OrigTN() string {
	return s.Payload.Orig.TN
}

func (s *sshaken_passport) DestTN() string {
	if len(s.Payload.Dest.TN) > 0 {
		return s.Payload.Dest.TN[0]
	}
	return ""
}

func (s *sshaken_passport) Iat() time.Time {
	return time.Unix(s.Payload.Iat, 0)
}

func (s *sshaken_passport) check_claims() error {
	if s.Header.Alg != "ES256" {
		return errors.New("'alg' value should be 'ES256'")
	}
	if s.Header.Ppt != "shaken" {
		return errors.New("'ppt' value should be 'shaken'")
	}
	if s.Header.Typ != "passport" {
		return errors.New("'typ' value should be 'passport'")
	}
	if s.Header.X5u == "" {
		return errors.New("'x5u' value should not be empty")
	}
	if s.Payload.Attest == "" {
		return errors.New("'attest' value should not be empty")
	}
	if len(s.Payload.Dest.TN) == 0 {
		return errors.New("dest tn value should not be empty")
	}
	if s.Payload.Iat == 0 {
		return errors.New("missing 'iat' claim")
	}
	if s.Payload.Orig.TN == "" {
		return errors.New("orig tn value should not be empty")
	}
	if s.Payload.Origid == "" {
		return errors.New("'origid' value should not be empty")
	}
	return nil
}

// Copyright (c) 2013 Kyle Isom <kyle@tyrfingr.is>
// Copyright (c) 2012 The Go Authors. All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//    * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//    * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//    * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package ecies

import (
	"bytes"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"testing"
)

var dumpEnc bool

func init() {
	flDump := flag.Bool("dump", false, "write encrypted test message to file")
	flag.Parse()
	dumpEnc = *flDump
}

type kdftest struct {
	key, s1 string
	kdlen   int
	output  string
}

var kdftests = []kdftest{
	{
		key:    "38f9a331c022f51d66658f301837108c9710d5ee0697bfac97bb6b0ea8d4f273",
		s1:     "",
		kdlen:  0,
		output: "",
	},
	{
		key:    "38f9a331c022f51d66658f301837108c9710d5ee0697bfac97bb6b0ea8d4f273",
		s1:     "",
		kdlen:  32,
		output: "bbfb3912ffc0d1789be7c3c2773fb6abd8df69578df2ca16beee3d0f7a9692d1",
	},
	{
		key:    "38f9a331c022f51d66658f301837108c9710d5ee0697bfac97bb6b0ea8d4f273",
		s1:     "",
		kdlen:  64,
		output: "bbfb3912ffc0d1789be7c3c2773fb6abd8df69578df2ca16beee3d0f7a9692d1eae543288220e41452942fe268297fff38423b65b19cf7c6263aa611a4190741",
	},
	{
		key:    "38f9a331c022f51d66658f301837108c9710d5ee0697bfac97bb6b0ea8d4f273",
		s1:     "",
		kdlen:  242,
		output: "bbfb3912ffc0d1789be7c3c2773fb6abd8df69578df2ca16beee3d0f7a9692d1eae543288220e41452942fe268297fff38423b65b19cf7c6263aa611a41907410a49acd5adbfbe93e902349105d7bd7ef5b106d9357b20bb4a7977d548bc2bf1a0b275a9f1de19ff8f963ec58171aa31da964edb0131436d88a7714e2429d85693409f718c8fea7ecaa076dce68f282aba4010a42feedcb1affc350497fa1078ad89e23e8a04ba2ef179c3c625b054817d792076b5882e80925d45a2874285d45a7767fa6a3853bd8417930923a554743f6eb4691e4790a97c467337a307c64a8bed5b5c1829d0a690a554d3e5334ee4980a",
	},
	{
		key:    "38f9a331c022f51d66658f301837108c9710d5ee0697bfac97bb6b0ea8d4f273",
		s1:     "343434",
		kdlen:  242,
		output: "71e6fe05a6a57e2312a996eb3b91fc613fb57196ea09880b7b0ed880afa399f942ad56c1a484f4ccd329f9c21911ae09b497e991d8f47060114c8f137ab65df10c3f1d2478a7af913aa406817f34ef55d7852f3390f8201056451b0c29fb73a5c9e5a5093e6d93fefa15846b81bc02e1a7033948d67fc70c1dbbbf11d97f34b15d9967709c666eff05d895cc2f415064b382a98e68c178f9e7e2d9b6169ef4cdd5bb855b21d0ff71339e7e5b3afcb2393a8c6f4c7a4e618094222be87cbfdd2417ebdced5870bfbe8761d2e646c5a62dd31852dba399507adb84334a81b4ce54714f1828a0cdf3067908815d516368c2dd58",
	},
}

func TestKDF(t *testing.T) {
	for _, test := range kdftests {
		z, _ := hex.DecodeString(test.key)
		s1, _ := hex.DecodeString(test.s1)
		h := sha256.New()
		k, err := ConcatKDF(h, z, s1, test.kdlen)
		if err != nil {
			t.Error(err)
		}
		if len(k) != test.kdlen {
			t.Errorf("output length mismatch: got %d, want %d", len(k), test.kdlen)
		}
		if hexk := hex.EncodeToString(k); hexk != test.output {
			t.Errorf("output mismatch:\ngot  %s\nwant %s", hexk, test.output)
		}
		if t.Failed() {
			t.Fatalf("failed test: %#v", test)
		}
	}
}

var skLen int
var ErrBadSharedKeys = fmt.Errorf("ecies: shared keys don't match")

// cmpParams compares a set of ECIES parameters. We assume, as per the
// docs, that AES is the only supported symmetric encryption algorithm.
func cmpParams(p1, p2 *ECIESParams) bool {
	if p1.hashAlgo != p2.hashAlgo {
		return false
	} else if p1.KeyLen != p2.KeyLen {
		return false
	} else if p1.BlockSize != p2.BlockSize {
		return false
	}
	return true
}

// cmpPublic returns true if the two public keys represent the same pojnt.
func cmpPublic(pub1, pub2 PublicKey) bool {
	if pub1.X == nil || pub1.Y == nil {
		fmt.Println(ErrInvalidPublicKey.Error())
		return false
	}
	if pub2.X == nil || pub2.Y == nil {
		fmt.Println(ErrInvalidPublicKey.Error())
		return false
	}
	pub1Out := elliptic.Marshal(pub1.Curve, pub1.X, pub1.Y)
	pub2Out := elliptic.Marshal(pub2.Curve, pub2.X, pub2.Y)

	return bytes.Equal(pub1Out, pub2Out)
}

// cmpPrivate returns true if the two private keys are the same.
func cmpPrivate(prv1, prv2 *PrivateKey) bool {
	if prv1 == nil || prv1.D == nil {
		return false
	} else if prv2 == nil || prv2.D == nil {
		return false
	} else if prv1.D.Cmp(prv2.D) != 0 {
		return false
	} else {
		return cmpPublic(prv1.PublicKey, prv2.PublicKey)
	}
}

// Validate the ECDH component.
func TestSharedKey(t *testing.T) {
	prv1, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}
	skLen = MaxSharedKeyLength(&prv1.PublicKey) / 2

	prv2, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	sk1, err := prv1.GenerateShared(&prv2.PublicKey, skLen, skLen)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	sk2, err := prv2.GenerateShared(&prv1.PublicKey, skLen, skLen)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if !bytes.Equal(sk1, sk2) {
		fmt.Println(ErrBadSharedKeys.Error())
		t.FailNow()
	}
}

// Verify that the key generation code fails when too much key data is
// requested.
func TestTooBigSharedKey(t *testing.T) {
	prv1, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	prv2, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	_, err = prv1.GenerateShared(&prv2.PublicKey, skLen*2, skLen*2)
	if err != ErrSharedKeyTooBig {
		fmt.Println("ecdh: shared key should be too large for curve")
		t.FailNow()
	}

	_, err = prv2.GenerateShared(&prv1.PublicKey, skLen*2, skLen*2)
	if err != ErrSharedKeyTooBig {
		fmt.Println("ecdh: shared key should be too large for curve")
		t.FailNow()
	}
}

// Ensure a public key can be successfully marshalled and unmarshalled, and
// that the decoded key is the same as the original.
func TestMarshalPublic(t *testing.T) {
	prv, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	out, err := MarshalPublic(&prv.PublicKey)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	pub, err := UnmarshalPublic(out)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if !cmpPublic(prv.PublicKey, *pub) {
		fmt.Println("ecies: failed to unmarshal public key")
		t.FailNow()
	}
}

// Ensure that a private key can be encoded into DER format, and that
// the resulting key is properly parsed back into a public key.
func TestMarshalPrivate(t *testing.T) {
	prv, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	out, err := MarshalPrivate(prv)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if dumpEnc {
		ioutil.WriteFile("test.out", out, 0644)
	}

	prv2, err := UnmarshalPrivate(out)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if !cmpPrivate(prv, prv2) {
		fmt.Println("ecdh: private key import failed")
		t.FailNow()
	}
}

// Ensure that a private key can be successfully encoded to PEM format, and
// the resulting key is properly parsed back in.
func TestPrivatePEM(t *testing.T) {
	prv, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	out, err := ExportPrivatePEM(prv)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if dumpEnc {
		ioutil.WriteFile("test.key", out, 0644)
	}

	prv2, err := ImportPrivatePEM(out)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	} else if !cmpPrivate(prv, prv2) {
		fmt.Println("ecdh: import from PEM failed")
		t.FailNow()
	}
}

// Ensure that a public key can be successfully encoded to PEM format, and
// the resulting key is properly parsed back in.
func TestPublicPEM(t *testing.T) {
	prv, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	out, err := ExportPublicPEM(&prv.PublicKey)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if dumpEnc {
		ioutil.WriteFile("test.pem", out, 0644)
	}

	pub2, err := ImportPublicPEM(out)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	} else if !cmpPublic(prv.PublicKey, *pub2) {
		fmt.Println("ecdh: import from PEM failed")
		t.FailNow()
	}
}

// Benchmark the generation of P256 keys.
func BenchmarkGenerateKeyP256(b *testing.B) {
	for i := 0; i < b.N; i++ {
		if _, err := GenerateKey(rand.Reader, elliptic.P256(), nil); err != nil {
			fmt.Println(err.Error())
			b.FailNow()
		}
	}
}

// Benchmark the generation of P256 shared keys.
func BenchmarkGenSharedKeyP256(b *testing.B) {
	prv, err := GenerateKey(rand.Reader, elliptic.P256(), nil)
	if err != nil {
		fmt.Println(err.Error())
		b.FailNow()
	}

	for i := 0; i < b.N; i++ {
		_, err := prv.GenerateShared(&prv.PublicKey, skLen, skLen)
		if err != nil {
			fmt.Println(err.Error())
			b.FailNow()
		}
	}
}

// Verify that an encrypted message can be successfully decrypted.
func TestEncryptDecrypt(t *testing.T) {
	prv1, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	prv2, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	message := []byte("Hello, world.")
	ct, err := Encrypt(rand.Reader, &prv2.PublicKey, message, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	pt, err := prv2.Decrypt(rand.Reader, ct, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if !bytes.Equal(pt, message) {
		fmt.Println("ecies: plaintext doesn't match message")
		t.FailNow()
	}

	_, err = prv1.Decrypt(rand.Reader, ct, nil, nil)
	if err == nil {
		fmt.Println("ecies: encryption should not have succeeded")
		t.FailNow()
	}
}

// TestMarshalEncryption validates the encode/decode produces a valid
// ECIES encryption key.
func TestMarshalEncryption(t *testing.T) {
	prv1, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	out, err := MarshalPrivate(prv1)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	prv2, err := UnmarshalPrivate(out)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	message := []byte("Hello, world.")
	ct, err := Encrypt(rand.Reader, &prv2.PublicKey, message, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	pt, err := prv2.Decrypt(rand.Reader, ct, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	if !bytes.Equal(pt, message) {
		fmt.Println("ecies: plaintext doesn't match message")
		t.FailNow()
	}

	_, err = prv1.Decrypt(rand.Reader, ct, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

}

type testCase struct {
	Curve    elliptic.Curve
	Name     string
	Expected bool
}

var testCases = []testCase{
	testCase{
		Curve:    elliptic.P256(),
		Name:     "P256",
		Expected: true,
	},
	testCase{
		Curve:    elliptic.P384(),
		Name:     "P384",
		Expected: true,
	},
	testCase{
		Curve:    elliptic.P521(),
		Name:     "P521",
		Expected: true,
	},
}

// Test parameter selection for each curve, and that P224 fails automatic
// parameter selection (see README for a discussion of P224). Ensures that
// selecting a set of parameters automatically for the given curve works.
func TestParamSelection(t *testing.T) {
	for _, c := range testCases {
		testParamSelection(t, c)
	}
}

func testParamSelection(t *testing.T, c testCase) {
	params := ParamsFromCurve(c.Curve)
	if params == nil && c.Expected {
		fmt.Printf("%s (%s)\n", ErrInvalidParams.Error(), c.Name)
		t.FailNow()
	} else if params != nil && !c.Expected {
		fmt.Printf("ecies: parameters should be invalid (%s)\n",
			c.Name)
		t.FailNow()
	}

	prv1, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Printf("%s (%s)\n", err.Error(), c.Name)
		t.FailNow()
	}

	prv2, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Printf("%s (%s)\n", err.Error(), c.Name)
		t.FailNow()
	}

	message := []byte("Hello, world.")
	ct, err := Encrypt(rand.Reader, &prv2.PublicKey, message, nil, nil)
	if err != nil {
		fmt.Printf("%s (%s)\n", err.Error(), c.Name)
		t.FailNow()
	}

	pt, err := prv2.Decrypt(rand.Reader, ct, nil, nil)
	if err != nil {
		fmt.Printf("%s (%s)\n", err.Error(), c.Name)
		t.FailNow()
	}

	if !bytes.Equal(pt, message) {
		fmt.Printf("ecies: plaintext doesn't match message (%s)\n",
			c.Name)
		t.FailNow()
	}

	_, err = prv1.Decrypt(rand.Reader, ct, nil, nil)
	if err == nil {
		fmt.Printf("ecies: encryption should not have succeeded (%s)\n",
			c.Name)
		t.FailNow()
	}

}

// Ensure that the basic public key validation in the decryption operation
// works.
func TestBasicKeyValidation(t *testing.T) {
	badBytes := []byte{0, 1, 5, 6, 7, 8, 9}

	prv, err := GenerateKey(rand.Reader, DefaultCurve, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	message := []byte("Hello, world.")
	ct, err := Encrypt(rand.Reader, &prv.PublicKey, message, nil, nil)
	if err != nil {
		fmt.Println(err.Error())
		t.FailNow()
	}

	for _, b := range badBytes {
		ct[0] = b
		_, err := prv.Decrypt(rand.Reader, ct, nil, nil)
		if err != ErrInvalidPublicKey {
			fmt.Println("ecies: validated an invalid key")
			t.FailNow()
		}
	}
}

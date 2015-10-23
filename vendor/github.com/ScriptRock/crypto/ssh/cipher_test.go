// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ssh

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"testing"
)

func TestDefaultCiphersExist(t *testing.T) {
	for _, cipherAlgo := range supportedCiphers {
		if _, ok := cipherModes[cipherAlgo]; !ok {
			t.Errorf("default cipher %q is unknown", cipherAlgo)
		}
	}
}

type kexAlgBundle struct {
	kr   *kexResult
	algs directionAlgorithms
}

func TestPacketCiphers(t *testing.T) {
	for cipher := range cipherModes {
		kexAlgs := []*kexAlgBundle{
			&kexAlgBundle{
				kr: &kexResult{Hash: crypto.SHA512},
				algs: directionAlgorithms{
					Cipher:      cipher,
					MAC:         "hmac-sha2-512",
					Compression: "none",
				},
			},
			&kexAlgBundle{
				kr: &kexResult{Hash: crypto.SHA256},
				algs: directionAlgorithms{
					Cipher:      cipher,
					MAC:         "hmac-sha2-256",
					Compression: "none",
				},
			},
			&kexAlgBundle{
				kr: &kexResult{Hash: crypto.SHA1},
				algs: directionAlgorithms{
					Cipher:      cipher,
					MAC:         "hmac-sha1",
					Compression: "none",
				},
			},
			&kexAlgBundle{
				kr: &kexResult{Hash: crypto.SHA1},
				algs: directionAlgorithms{
					Cipher:      cipher,
					MAC:         "hmac-sha1-96",
					Compression: "none",
				},
			},
			&kexAlgBundle{
				kr: &kexResult{Hash: crypto.MD5},
				algs: directionAlgorithms{
					Cipher:      cipher,
					MAC:         "hmac-md5",
					Compression: "none",
				},
			},
		}
		for _, kexAlg := range kexAlgs {
			t.Logf("cipher %v kex %v", cipher, kexAlg.algs.MAC)

			kr := kexAlg.kr
			algs := kexAlg.algs
			client, err := newPacketCipher(encrypt, clientKeys, algs, kr)
			if err != nil {
				t.Errorf("newPacketCipher(client, %q): %v", cipher, err)
				continue
			}
			server, err := newPacketCipher(decrypt, clientKeys, algs, kr)
			if err != nil {
				t.Errorf("newPacketCipher(client, %q): %v", cipher, err)
				continue
			}

			want := "bla bla"
			input := []byte(want)
			buf := &bytes.Buffer{}
			if err := client.writePacket(0, buf, rand.Reader, input); err != nil {
				t.Errorf("writePacket(%q): %v", cipher, err)
				continue
			}

			packet, err := server.readPacket(0, buf)
			if err != nil {
				t.Errorf("readPacket(%q): %v", cipher, err)
				continue
			}

			if string(packet) != want {
				t.Errorf("roundtrip(%q): got %q, want %q", cipher, packet, want)
			}
		}
	}
}

func TestCBCOracleCounterMeasure(t *testing.T) {
	kr := &kexResult{Hash: crypto.SHA1}
	algs := directionAlgorithms{
		Cipher:      "aes128-cbc",
		MAC:         "hmac-sha1",
		Compression: "none",
	}
	client, err := newPacketCipher(encrypt, clientKeys, algs, kr)
	if err != nil {
		t.Fatalf("newPacketCipher(client): %v", err)
	}

	want := "bla bla"
	input := []byte(want)
	buf := &bytes.Buffer{}
	if err := client.writePacket(0, buf, rand.Reader, input); err != nil {
		t.Errorf("writePacket: %v", err)
	}

	packetSize := buf.Len()
	buf.Write(make([]byte, 2*maxPacket))

	// We corrupt each byte, but this usually will only test the
	// 'packet too large' or 'MAC failure' cases.
	lastRead := -1
	for i := 0; i < packetSize; i++ {
		server, err := newPacketCipher(decrypt, clientKeys, algs, kr)
		if err != nil {
			t.Fatalf("newPacketCipher(client): %v", err)
		}

		fresh := &bytes.Buffer{}
		fresh.Write(buf.Bytes())
		fresh.Bytes()[i] ^= 0x01

		before := fresh.Len()
		_, err = server.readPacket(0, fresh)
		if err == nil {
			t.Errorf("corrupt byte %d: readPacket succeeded ", i)
			continue
		}
		if _, ok := err.(cbcError); !ok {
			t.Errorf("corrupt byte %d: got %v (%T), want cbcError", i, err, err)
			continue
		}

		after := fresh.Len()
		bytesRead := before - after
		if bytesRead < maxPacket {
			t.Errorf("corrupt byte %d: read %d bytes, want more than %d", i, bytesRead, maxPacket)
			continue
		}

		if i > 0 && bytesRead != lastRead {
			t.Errorf("corrupt byte %d: read %d bytes, want %d bytes read", i, bytesRead, lastRead)
		}
		lastRead = bytesRead
	}
}

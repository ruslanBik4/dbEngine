package excel

import (
	"bytes"
	"crypto/md5"
	"crypto/rc4"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"unicode/utf16"
)

var (
	// ErrPasswordRequired is returned for an encrypted workbook opened without a password.
	ErrPasswordRequired = errors.New("workbook is password protected: use ExcelToPostgresWithPassword")
	// ErrWrongPassword is returned when the password does not match.
	ErrWrongPassword = errors.New("wrong password")
)

// defaultXLSPassword is what Excel uses for files that are only
// "write protected" (encrypted, but open without asking for a password).
const defaultXLSPassword = "VelvetSweatshop"

func utf16le(s string) []byte {
	u := utf16.Encode([]rune(s))
	b := make([]byte, len(u)*2)
	for i, c := range u {
		binary.LittleEndian.PutUint16(b[i*2:], c)
	}
	return b
}

// rc4KeyFunc returns the RC4 key for a 1024-byte block.
type rc4KeyFunc func(block uint32) []byte

// filePassKey parses a FILEPASS record and derives the key for password.
func filePassKey(fp []byte, password string) (rc4KeyFunc, error) {
	if len(fp) < 2 {
		return nil, fmt.Errorf("xls: bad FILEPASS record")
	}
	if binary.LittleEndian.Uint16(fp) == 0 {
		return nil, fmt.Errorf("xls: XOR obfuscation is not supported")
	}
	if len(fp) < 6 {
		return nil, fmt.Errorf("xls: bad FILEPASS record")
	}
	major, minor := binary.LittleEndian.Uint16(fp[2:]), binary.LittleEndian.Uint16(fp[4:])
	pw := utf16le(password)
	blockLE := func(b uint32) []byte { x := make([]byte, 4); binary.LittleEndian.PutUint32(x, b); return x }

	switch {
	case major == 1 && minor == 1: // Office 97/2000 binary RC4 (MS-OFFCRYPTO 2.3.6)
		if len(fp) < 6+48 {
			return nil, fmt.Errorf("xls: bad RC4 header")
		}
		salt, encVer, encHash := fp[6:22], fp[22:38], fp[38:54]
		h0 := md5.Sum(pw)
		var buf bytes.Buffer
		for i := 0; i < 16; i++ {
			buf.Write(h0[:5])
			buf.Write(salt)
		}
		h1 := md5.Sum(buf.Bytes())
		key := func(b uint32) []byte {
			k := md5.Sum(append(append([]byte{}, h1[:5]...), blockLE(b)...))
			return k[:]
		}
		c, _ := rc4.NewCipher(key(0))
		ver := make([]byte, 16)
		hash := make([]byte, 16)
		c.XORKeyStream(ver, encVer)
		c.XORKeyStream(hash, encHash)
		if sum := md5.Sum(ver); !bytes.Equal(sum[:], hash) {
			return nil, ErrWrongPassword
		}
		return key, nil

	case (major == 2 || major == 3 || major == 4) && minor == 2: // RC4 CryptoAPI (MS-OFFCRYPTO 2.3.5)
		p := 6
		if len(fp) < p+8 {
			return nil, fmt.Errorf("xls: bad CryptoAPI header")
		}
		hdrSize := int(binary.LittleEndian.Uint32(fp[p+4:]))
		p += 8
		if len(fp) < p+hdrSize || hdrSize < 32 {
			return nil, fmt.Errorf("xls: bad CryptoAPI header")
		}
		keyBits := int(binary.LittleEndian.Uint32(fp[p+16:]))
		if keyBits == 0 {
			keyBits = 40
		}
		p += hdrSize
		if len(fp) < p+4+16+16+4+20 {
			return nil, fmt.Errorf("xls: bad CryptoAPI verifier")
		}
		saltSize := int(binary.LittleEndian.Uint32(fp[p:]))
		if saltSize != 16 {
			return nil, fmt.Errorf("xls: unexpected salt size %d", saltSize)
		}
		salt, encVer := fp[p+4:p+20], fp[p+20:p+36]
		encHash := fp[p+40 : p+60]
		h0 := sha1.Sum(append(append([]byte{}, salt...), pw...))
		keyLen := keyBits / 8
		key := func(b uint32) []byte {
			h := sha1.Sum(append(append([]byte{}, h0[:]...), blockLE(b)...))
			k := h[:keyLen]
			if keyBits == 40 { // 40-bit keys are zero padded to 128 bits
				k = append(append([]byte{}, k...), make([]byte, 11)...)
			}
			return k
		}
		c, _ := rc4.NewCipher(key(0))
		ver := make([]byte, 16)
		hash := make([]byte, 20)
		c.XORKeyStream(ver, encVer)
		c.XORKeyStream(hash, encHash)
		if sum := sha1.Sum(ver); !bytes.Equal(sum[:], hash) {
			return nil, ErrWrongPassword
		}
		return key, nil
	}
	return nil, fmt.Errorf("xls: unsupported encryption %d.%d", major, minor)
}

// records that are never encrypted (MS-XLS 2.2.10)
var plainRecords = map[uint16]bool{
	0x0809: true, // BOF
	0x002F: true, // FILEPASS
	0x0194: true, // UsrExcl
	0x0195: true, // FileLock
	0x00E1: true, // InterfaceHdr
	0x0196: true, // RRDInfo
	0x0138: true, // RRDHead
}

// decryptBIFF decrypts, in place, every record that follows FILEPASS. The
// RC4 key changes every 1024 bytes of the stream; record headers are not
// encrypted but still consume key stream.
func decryptBIFF(stream []byte, recs []biffRec, fpIndex int, key rc4KeyFunc) {
	var (
		curBlock uint32 = ^uint32(0)
		ks              = make([]byte, 1024)
	)
	xor := func(from, to int) {
		for p := from; p < to; p++ {
			b := uint32(p / 1024)
			if b != curBlock {
				c, _ := rc4.NewCipher(key(b))
				for i := range ks {
					ks[i] = 0
				}
				c.XORKeyStream(ks, ks)
				curBlock = b
			}
			stream[p] ^= ks[p%1024]
		}
	}
	for _, r := range recs[fpIndex+1:] {
		start, end := r.pos+4, r.pos+4+len(r.data)
		switch {
		case plainRecords[r.id]:
		case r.id == recBoundSheet: // lbPlyPos (stream offset) is stored in clear
			xor(start+4, end)
		default:
			xor(start, end)
		}
	}
}

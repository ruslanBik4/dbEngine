package excel

import (
	"bytes"
	"errors"
	_ "image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// testdata/images: b.* - a 64x48 green bitmap as EMF (EMR_STRETCHDIBITS),
// WMF (META_STRETCHDIB), BMP, TIFF; v.emf - vector drawing only.
func TestWebImage(t *testing.T) {
	for _, name := range []string{"b.emf", "b.wmf", "b.bmp", "b.tif"} {
		t.Run(name, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("testdata", "images", name))
			if err != nil {
				t.Fatal(err)
			}
			out, ext, err := webImage(data, filepath.Ext(name))
			if err != nil {
				t.Fatal(err)
			}
			if ext != ".png" {
				t.Fatalf("ext %s", ext)
			}
			img, err := png.Decode(bytes.NewReader(out))
			if err != nil {
				t.Fatal(err)
			}
			if b := img.Bounds(); b.Dx() != 64 || b.Dy() != 48 {
				t.Errorf("size %v", b)
			}
			r, g, b, a := img.At(10, 10).RGBA()
			if r>>8 != 10 || g>>8 != 200 || b>>8 != 30 || a>>8 != 255 {
				t.Errorf("pixel %d %d %d %d", r>>8, g>>8, b>>8, a>>8)
			}
		})
	}

	data, err := os.ReadFile(filepath.Join("testdata", "images", "v.emf"))
	if err != nil {
		t.Fatal(err)
	}
	out, ext, err := webImage(data, ".emf")
	if !errors.Is(err, errVectorMetafile) || ext != ".emf" || !bytes.Equal(out, data) {
		t.Errorf("vector EMF: %s %v", ext, err)
	}

	png1 := []byte("\x89PNG\r\n\x1a\nrest")
	if out, ext, err := webImage(png1, ".png"); err != nil || ext != ".png" || !bytes.Equal(out, png1) {
		t.Errorf("png changed: %s %v", ext, err)
	}
}

func TestDecodeDIBTopDownAndPalette(t *testing.T) {
	// 2x2, 8 bpp, top-down, palette of 2
	hdr := make([]byte, 40)
	put32 := func(b []byte, off int, v int32) {
		b[off], b[off+1], b[off+2], b[off+3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
	}
	put32(hdr, 0, 40)
	put32(hdr, 4, 2)
	put32(hdr, 8, -2)
	hdr[12], hdr[14] = 1, 8
	put32(hdr, 32, 2)
	bmi := append(hdr, 0, 0, 255, 0, 255, 0, 0, 0) // red, blue
	bits := []byte{0, 1, 0, 0, 1, 0, 0, 0}         // rows padded to 4 bytes
	img, _, _, err := decodeDIB(bmi, bits)
	if err != nil {
		t.Fatal(err)
	}
	at := func(x, y int) [3]uint32 {
		r, g, b, _ := img.At(x, y).RGBA()
		return [3]uint32{r >> 8, g >> 8, b >> 8}
	}
	if at(0, 0) != [3]uint32{255, 0, 0} || at(1, 0) != [3]uint32{0, 0, 255} || at(0, 1) != [3]uint32{0, 0, 255} {
		t.Errorf("pixels %v %v %v", at(0, 0), at(1, 0), at(0, 1))
	}
}

func TestVectorEMFWithLibreOffice(t *testing.T) {
	soffice, err := exec.LookPath("soffice")
	if err != nil {
		t.Skip(err)
	}
	data, err := os.ReadFile(filepath.Join("testdata", "images", "v.emf"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := sofficeToPNG(soffice, data, ".emf")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := png.Decode(bytes.NewReader(out)); err != nil {
		t.Fatal(err)
	}
}

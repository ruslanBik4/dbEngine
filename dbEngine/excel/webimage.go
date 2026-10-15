package excel

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/tiff"
)

// Browsers show PNG, JPEG, GIF, WebP, SVG (and BMP), but not the Windows
// metafiles (EMF/WMF) Excel uses for pasted pictures, nor TIFF in most
// browsers. webImage turns such pictures into PNG:
//   - EMF / WMF: almost always a wrapper around a bitmap (EMR_STRETCHDIBITS,
//     META_STRETCHDIB..., or an EMF+ image object): the bitmap is taken out
//     (as is when it is an embedded PNG/JPEG); several bitmaps - the largest.
//     A metafile with vector drawing only is returned as is with
//     errVectorMetafile (see sofficeToPNG);
//   - BMP, TIFF: re-encoded as PNG.
//
// Anything else is returned unchanged.
func webImage(data []byte, ext string) ([]byte, string, error) {
	switch strings.ToLower(ext) {
	case ".emf", ".wmf":
		out, outExt, err := metafileBitmap(data, strings.ToLower(ext))
		if err != nil {
			return data, ext, err
		}
		return out, outExt, nil

	case ".bmp", ".dib":
		b := data
		if len(b) > 14 && b[0] == 'B' && b[1] == 'M' {
			b = b[14:]
		}
		img, raw, rawExt, err := decodeDIB(b, nil)
		switch {
		case err != nil:
			return data, ext, err
		case raw != nil:
			return raw, rawExt, nil
		}
		return encodePNG(img, data, ext)

	case ".tif", ".tiff":
		img, err := tiff.Decode(bytes.NewReader(data))
		if err != nil {
			return data, ext, err
		}
		return encodePNG(img, data, ext)
	}
	return data, ext, nil
}

var errVectorMetafile = errors.New("vector metafile without a bitmap, kept as is (browsers don't show it)")

func encodePNG(img image.Image, orig []byte, ext string) ([]byte, string, error) {
	var buf bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, img); err != nil {
		return orig, ext, err
	}
	return buf.Bytes(), ".png", nil
}

// metaBitmap is a picture found in a metafile: decoded, or an embedded file.
type metaBitmap struct {
	img    image.Image
	raw    []byte
	rawExt string
	area   int
}

func (m *metaBitmap) better(o *metaBitmap) bool { return m == nil || o.area > m.area }

func metafileBitmap(data []byte, ext string) ([]byte, string, error) {
	var best *metaBitmap
	var err error
	if ext == ".wmf" || (len(data) > 4 && binary.LittleEndian.Uint32(data) == 0x9AC6CDD7) {
		best, err = wmfBitmaps(data)
	} else {
		best, err = emfBitmaps(data, 0)
	}
	if err != nil {
		return nil, "", err
	}
	if best == nil {
		return nil, "", errVectorMetafile
	}
	if best.raw != nil {
		return best.raw, best.rawExt, nil
	}
	out, outExt, err := encodePNG(best.img, nil, "")
	return out, outExt, err
}

func le16(b []byte, off int) int { return int(binary.LittleEndian.Uint16(b[off:])) }
func le32(b []byte, off int) int { return int(binary.LittleEndian.Uint32(b[off:])) }

// emfBitmaps walks the EMF records and returns the largest bitmap.
func emfBitmaps(data []byte, depth int) (*metaBitmap, error) {
	if len(data) < 88 || le32(data, 0) != 1 || string(data[40:44]) != " EMF" {
		return nil, errors.New("not an EMF file")
	}
	var best *metaBitmap
	consider := func(m *metaBitmap) {
		if m != nil && best.better(m) {
			best = m
		}
	}
	// bitmap record -> offsets of offBmiSrc (the next 3 dwords are cbBmiSrc, offBitsSrc, cbBitsSrc)
	bmiAt := map[int]int{
		76:  84, // EMR_BITBLT
		77:  84, // EMR_STRETCHBLT
		114: 84, // EMR_ALPHABLEND
		116: 84, // EMR_TRANSPARENTBLT
		80:  48, // EMR_SETDIBITSTODEVICE
		81:  48, // EMR_STRETCHDIBITS
	}
	var plus emfPlusObjects
	for off := 0; off+8 <= len(data); {
		typ, size := le32(data, off), le32(data, off+4)
		if size < 8 || off+size > len(data) {
			break
		}
		rec := data[off : off+size]
		if at, ok := bmiAt[typ]; ok && at+16 <= len(rec) {
			offBmi, cbBmi, offBits, cbBits := le32(rec, at), le32(rec, at+4), le32(rec, at+8), le32(rec, at+12)
			if cbBmi > 0 && offBmi+cbBmi <= len(rec) && offBits+cbBits <= len(rec) {
				consider(dibBitmap(rec[offBmi:offBmi+cbBmi], rec[offBits:offBits+cbBits]))
			}
		}
		if typ == 70 && len(rec) >= 16 && le32(rec, 12) == 0x2B464D45 { // EMR_COMMENT "EMF+"
			plus.add(rec[16 : 12+min(le32(rec, 8), len(rec)-12)])
		}
		if typ == 14 { // EMR_EOF
			break
		}
		off += size
	}
	for _, obj := range plus.images() {
		consider(emfPlusImage(obj, depth))
	}
	return best, nil
}

// emfPlusObjects collects EMF+ image objects (EmfPlusObject, ObjectType
// Image), joining the parts of objects split over several records.
type emfPlusObjects struct {
	parts map[int][]byte
	total map[int]int
	done  [][]byte
}

func (p *emfPlusObjects) add(b []byte) {
	if p.parts == nil {
		p.parts, p.total = map[int][]byte{}, map[int]int{}
	}
	for off := 0; off+12 <= len(b); {
		typ, flags, size, dataSize := le16(b, off), le16(b, off+2), le32(b, off+4), le32(b, off+8)
		if size < 12 || off+size > len(b) || 12+dataSize > size {
			return
		}
		rec := b[off+12 : off+12+dataSize]
		if typ == 0x4008 && (flags>>8)&0x7F == 5 { // EmfPlusObject, ObjectTypeImage
			id := flags & 0xFF
			if flags&0x8000 != 0 { // continued: TotalObjectSize, then a part
				if len(rec) >= 4 {
					p.total[id] = le32(rec, 0)
					p.parts[id] = append(p.parts[id], rec[4:]...)
					if len(p.parts[id]) >= p.total[id] {
						p.done = append(p.done, p.parts[id])
						delete(p.parts, id)
					}
				}
			} else if len(p.parts[id]) > 0 { // last part of a continued object
				p.done = append(p.done, append(p.parts[id], rec...))
				delete(p.parts, id)
			} else {
				p.done = append(p.done, rec)
			}
		}
		off += size
	}
}

func (p *emfPlusObjects) images() [][]byte { return p.done }

// emfPlusImage decodes an EmfPlusImage object.
func emfPlusImage(b []byte, depth int) *metaBitmap {
	if len(b) < 8 {
		return nil
	}
	switch le32(b, 4) {
	case 1: // ImageDataTypeBitmap
		if len(b) < 28 {
			return nil
		}
		w, h, stride, pixFmt, kind := le32(b, 8), le32(b, 12), int(int32(le32(b, 16))), le32(b, 20), le32(b, 24)
		data := b[28:]
		if kind == 1 { // compressed: a PNG, JPEG, GIF... file
			if ext := sniffImageExt(data); ext != ".bin" {
				return &metaBitmap{raw: data, rawExt: ext, area: w * h}
			}
			return nil
		}
		return &metaBitmap{img: emfPlusPixels(data, w, h, stride, pixFmt), area: w * h}
	case 2: // ImageDataTypeMetafile: Type, MetafileDataSize, data
		if len(b) < 16 || depth > 2 {
			return nil
		}
		data := b[16:min(len(b), 16+le32(b, 12))]
		var m *metaBitmap
		switch le32(b, 8) {
		case 1, 2: // WMF, WMF placeable
			m, _ = wmfBitmaps(data)
		default: // EMF, EMF+ only, EMF+ dual
			m, _ = emfBitmaps(data, depth+1)
		}
		return m
	}
	return nil
}

// emfPlusPixels decodes uncompressed EMF+ bitmaps of the common 24/32 bpp
// pixel formats; nil for others.
func emfPlusPixels(b []byte, w, h, stride, pixFmt int) image.Image {
	if w <= 0 || h <= 0 || w > 1<<15 || h > 1<<15 {
		return nil
	}
	var bpp int
	alpha := false
	switch pixFmt {
	case 0x00021808: // PixelFormat24bppRGB
		bpp = 3
	case 0x00022009: // PixelFormat32bppRGB
		bpp = 4
	case 0x0026200A, 0x000E200B: // PixelFormat32bppARGB, PARGB
		bpp, alpha = 4, true
	default:
		return nil
	}
	if stride == 0 {
		stride = (w*bpp + 3) &^ 3
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		row := y * stride
		if stride < 0 {
			row = (h - 1 - y) * -stride
		}
		if row < 0 || row+w*bpp > len(b) {
			return nil
		}
		for x := 0; x < w; x++ {
			p := b[row+x*bpp:]
			a := uint8(255)
			if alpha {
				a = p[3]
			}
			img.SetNRGBA(x, y, color.NRGBA{R: p[2], G: p[1], B: p[0], A: a})
		}
	}
	return img
}

// wmfBitmaps walks the WMF records and returns the largest bitmap.
func wmfBitmaps(data []byte) (*metaBitmap, error) {
	b := data
	if len(b) >= 22 && le32(b, 0) == 0x9AC6CDD7 { // META_PLACEABLE
		b = b[22:]
	}
	if len(b) < 18 || le16(b, 2) != 9 {
		return nil, errors.New("not a WMF file")
	}
	var best *metaBitmap
	// record function -> bytes of parameters before the DIB
	dibAt := map[int]int{
		0x0F43: 22, // META_STRETCHDIB
		0x0B41: 20, // META_DIBSTRETCHBLT
		0x0940: 16, // META_DIBBITBLT
		0x0D33: 18, // META_SETDIBTODEV
	}
	for off := 18; off+6 <= len(b); {
		size, fn := le32(b, off)*2, le16(b, off+4)
		if size < 6 || off+size > len(b) {
			break
		}
		if fn == 0 { // META_EOF
			break
		}
		if at, ok := dibAt[fn]; ok && 6+at < size {
			if m := dibBitmap(b[off+6+at:off+size], nil); m != nil && best.better(m) {
				best = m
			}
		}
		off += size
	}
	return best, nil
}

func dibBitmap(bmi, bits []byte) *metaBitmap {
	img, raw, rawExt, err := decodeDIB(bmi, bits)
	switch {
	case err != nil:
		return nil
	case raw != nil:
		w, h := dibSize(bmi)
		return &metaBitmap{raw: raw, rawExt: rawExt, area: w * h}
	}
	r := img.Bounds()
	return &metaBitmap{img: img, area: r.Dx() * r.Dy()}
}

func dibSize(bmi []byte) (int, int) {
	if len(bmi) >= 12 && le32(bmi, 0) == 12 {
		return le16(bmi, 4), le16(bmi, 6)
	}
	if len(bmi) >= 12 {
		w, h := int(int32(le32(bmi, 4))), int(int32(le32(bmi, 8)))
		return w, max(h, -h)
	}
	return 0, 0
}

// decodeDIB decodes a device independent bitmap: bmi is BITMAPINFO (header,
// masks, palette); bits the pixels - nil when they follow the palette (a
// packed DIB). BI_JPEG / BI_PNG bitmaps are returned as raw files.
func decodeDIB(bmi, bits []byte) (image.Image, []byte, string, error) {
	if len(bmi) < 12 {
		return nil, nil, "", errors.New("DIB header too short")
	}
	hdr := le32(bmi, 0)
	var w, h, bpp, compression, clrUsed int
	topDown := false
	palEntry := 4
	if hdr == 12 { // BITMAPCOREHEADER
		w, h, bpp, palEntry = le16(bmi, 4), le16(bmi, 6), le16(bmi, 10), 3
	} else {
		if hdr < 40 || len(bmi) < 40 {
			return nil, nil, "", errors.New("unknown DIB header")
		}
		w, h = int(int32(le32(bmi, 4))), int(int32(le32(bmi, 8)))
		bpp, compression, clrUsed = le16(bmi, 14), le32(bmi, 16), le32(bmi, 32)
		if h < 0 {
			h, topDown = -h, true
		}
	}
	if w <= 0 || h <= 0 || w > 1<<15 || h > 1<<15 {
		return nil, nil, "", fmt.Errorf("bad DIB size %dx%d", w, h)
	}

	pos := hdr
	masks := [4]uint32{}
	switch {
	case compression == 3 || compression == 6: // BI_BITFIELDS, BI_ALPHABITFIELDS
		n := 3
		if compression == 6 {
			n = 4
		}
		src := hdr
		if hdr >= 52 { // V2+ headers hold the masks themselves
			src = 40
		} else {
			pos += n * 4
		}
		if src+n*4 > len(bmi) {
			return nil, nil, "", errors.New("DIB masks missing")
		}
		for i := 0; i < n; i++ {
			masks[i] = uint32(le32(bmi, src+i*4))
		}
		if hdr >= 56 && compression == 3 {
			masks[3] = uint32(le32(bmi, 52))
		}
	case bpp == 16:
		masks = [4]uint32{0x7C00, 0x03E0, 0x001F, 0}
	}

	nPal := clrUsed
	if nPal == 0 && bpp <= 8 {
		nPal = 1 << bpp
	}
	if bpp > 8 && hdr == 12 {
		nPal = 0
	}
	var pal []color.NRGBA
	for i := 0; i < nPal && pos+palEntry <= len(bmi); i++ {
		pal = append(pal, color.NRGBA{R: bmi[pos+2], G: bmi[pos+1], B: bmi[pos], A: 255})
		pos += palEntry
	}
	if bits == nil {
		if bpp <= 8 && len(pal) < nPal { // palette cut short
			return nil, nil, "", errors.New("DIB palette missing")
		}
		bits = bmi[min(pos, len(bmi)):]
	}

	switch compression {
	case 4: // BI_JPEG
		return nil, bits, ".jpeg", nil
	case 5: // BI_PNG
		return nil, bits, ".png", nil
	case 1, 2: // BI_RLE8, BI_RLE4
		return decodeRLE(bits, w, h, compression == 2, pal)
	case 0, 3, 6:
	default:
		return nil, nil, "", fmt.Errorf("DIB compression %d not supported", compression)
	}

	stride := (w*bpp + 31) / 32 * 4
	if len(bits) < stride*(h-1)+(w*bpp+7)/8 {
		return nil, nil, "", errors.New("DIB pixels missing")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	useAlpha := false
	if bpp == 32 && compression == 0 { // alpha only when the bitmap really has it
		for y := 0; y < h && !useAlpha; y++ {
			row := bits[y*stride:]
			for x := 0; x < w; x++ {
				if row[x*4+3] != 0 {
					useAlpha = true
					break
				}
			}
		}
	}
	palAt := func(i int) color.NRGBA {
		if i < len(pal) {
			return pal[i]
		}
		return color.NRGBA{A: 255}
	}
	for y := 0; y < h; y++ {
		row := bits[y*stride:]
		dy := h - 1 - y
		if topDown {
			dy = y
		}
		for x := 0; x < w; x++ {
			var c color.NRGBA
			switch bpp {
			case 1:
				c = palAt(int(row[x/8]>>(7-uint(x%8))) & 1)
			case 2:
				c = palAt(int(row[x/4]>>(6-2*uint(x%4))) & 3)
			case 4:
				c = palAt(int(row[x/2]>>(4-4*uint(x%2))) & 15)
			case 8:
				c = palAt(int(row[x]))
			case 16:
				c = fromMasks(uint32(binary.LittleEndian.Uint16(row[x*2:])), masks)
			case 24:
				p := row[x*3:]
				c = color.NRGBA{R: p[2], G: p[1], B: p[0], A: 255}
			case 32:
				if compression == 0 {
					p := row[x*4:]
					c = color.NRGBA{R: p[2], G: p[1], B: p[0], A: 255}
					if useAlpha {
						c.A = p[3]
					}
				} else {
					c = fromMasks(binary.LittleEndian.Uint32(row[x*4:]), masks)
				}
			default:
				return nil, nil, "", fmt.Errorf("DIB with %d bits per pixel not supported", bpp)
			}
			img.SetNRGBA(x, dy, c)
		}
	}
	return img, nil, "", nil
}

func fromMasks(v uint32, m [4]uint32) color.NRGBA {
	ch := func(mask uint32) uint8 {
		if mask == 0 {
			return 255
		}
		shift := 0
		for mask&1 == 0 {
			mask >>= 1
			shift++
		}
		x := (v >> uint(shift)) & mask
		return uint8(x * 255 / mask)
	}
	c := color.NRGBA{R: ch(m[0]), G: ch(m[1]), B: ch(m[2]), A: 255}
	if m[3] != 0 {
		c.A = ch(m[3])
	}
	return c
}

// decodeRLE decodes BI_RLE8 / BI_RLE4 bitmaps (always bottom-up).
func decodeRLE(b []byte, w, h int, rle4 bool, pal []color.NRGBA) (image.Image, []byte, string, error) {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	x, y := 0, 0
	set := func(i int) {
		if x < w && y < h {
			c := color.NRGBA{A: 255}
			if i < len(pal) {
				c = pal[i]
			}
			img.SetNRGBA(x, h-1-y, c)
		}
		x++
	}
	for p := 0; p+1 < len(b); {
		n, v := int(b[p]), b[p+1]
		p += 2
		if n > 0 { // encoded run
			for i := 0; i < n; i++ {
				if rle4 {
					set(int(v>>(4*uint(1-i%2))) & 15)
				} else {
					set(int(v))
				}
			}
			continue
		}
		switch v {
		case 0: // end of line
			x, y = 0, y+1
		case 1: // end of bitmap
			return img, nil, "", nil
		case 2: // delta
			if p+1 >= len(b) {
				return img, nil, "", nil
			}
			x, y = x+int(b[p]), y+int(b[p+1])
			p += 2
		default: // absolute run of v pixels, padded to 16 bits
			n := int(v)
			size := n
			if rle4 {
				size = (n + 1) / 2
			}
			if p+size > len(b) {
				return img, nil, "", nil
			}
			for i := 0; i < n; i++ {
				if rle4 {
					set(int(b[p+i/2]>>(4*uint(1-i%2))) & 15)
				} else {
					set(int(b[p+i]))
				}
			}
			p += size + size%2
		}
	}
	return img, nil, "", nil
}

// sofficeToPNG renders a vector metafile with LibreOffice.
func sofficeToPNG(soffice string, data []byte, ext string) ([]byte, error) {
	dir, err := os.MkdirTemp("", "dbengine-emf-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	src := filepath.Join(dir, "picture"+ext)
	if err := os.WriteFile(src, data, 0o600); err != nil {
		return nil, err
	}
	cmd := exec.Command(soffice, "--headless", "--norestore",
		"-env:UserInstallation=file://"+filepath.ToSlash(filepath.Join(dir, "profile")),
		"--convert-to", "png", "--outdir", dir, src)
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil {
			return nil, err
		}
	case <-time.After(60 * time.Second):
		_ = cmd.Process.Kill()
		return nil, errors.New("timeout")
	}
	return os.ReadFile(filepath.Join(dir, "picture.png"))
}

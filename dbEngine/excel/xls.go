package excel

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"slices"
	"strings"
	"unicode/utf16"

	"github.com/richardlehane/mscfb"
)

// errNeedConversion means the file is a format we don't parse natively
// (e.g. BIFF5/Excel 95) and should be converted with LibreOffice.
var errNeedConversion = errors.New("format requires conversion")

// errEncryptedOOXML: an OLE container holding an encrypted .xlsx package.
var errEncryptedOOXML = errors.New("encrypted OOXML package")

// BIFF record ids.
const (
	recBOF           = 0x0809
	recEOF           = 0x000A
	recBoundSheet    = 0x0085
	recDateMode      = 0x0022
	recSST           = 0x00FC
	recContinue      = 0x003C
	recLabelSST      = 0x00FD
	recLabel         = 0x0204
	recRString       = 0x00D6
	recNumber        = 0x0203
	recRK            = 0x027E
	recMulRK         = 0x00BD
	recBoolErr       = 0x0205
	recFormula       = 0x0006
	recString        = 0x0207
	recMergeCells    = 0x00E5
	recXF            = 0x00E0
	recFormat        = 0x041E
	recFilePass      = 0x002F
	recMsoDrawingGrp = 0x00EB
	recMsoDrawing    = 0x00EC
	recFont          = 0x0031
)

type biffRec struct {
	id   uint16
	data []byte
	pos  int // offset of record header in stream
}

func readXLS(data []byte, password string) (*Workbook, error) {
	doc, err := mscfb.New(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("xls: not an OLE2 file: %w", err)
	}
	var stream []byte
	for e, err := doc.Next(); err == nil; e, err = doc.Next() {
		switch e.Name {
		case "Workbook", "WORKBOOK", "workbook":
			stream, err = io.ReadAll(e)
			if err != nil {
				return nil, err
			}
		case "EncryptedPackage":
			return nil, errEncryptedOOXML // password protected .xlsx/.xlsm/.xlsb
		case "Book", "BOOK":
			if stream == nil {
				return nil, errNeedConversion // BIFF5 / Excel 5-95
			}
		}
	}
	if stream == nil {
		return nil, fmt.Errorf("xls: Workbook stream not found")
	}
	return parseBIFF8(stream, password)
}

func splitRecords(b []byte) []biffRec {
	var out []biffRec
	for p := 0; p+4 <= len(b); {
		id := binary.LittleEndian.Uint16(b[p:])
		n := int(binary.LittleEndian.Uint16(b[p+2:]))
		end := min(p+4+n, len(b))
		out = append(out, biffRec{id: id, data: b[p+4 : end], pos: p})
		p = end
	}
	return out
}

type xlsSheetRef struct {
	name   string
	pos    int
	hidden bool
}

func parseBIFF8(stream []byte, password string) (*Workbook, error) {
	recs := splitRecords(stream)
	if len(recs) == 0 || recs[0].id != recBOF || len(recs[0].data) < 4 {
		return nil, fmt.Errorf("xls: bad BOF")
	}
	if binary.LittleEndian.Uint16(recs[0].data) != 0x0600 {
		return nil, errNeedConversion // BIFF2-5
	}

	var (
		sheets   []xlsSheetRef
		sst      []string
		xfFmt    []uint16
		formats  = map[uint16]string{}
		date1904 bool
		dggData  []byte
		xfRaw    [][]byte
		fonts    [][]byte
	)
	// --- globals substream ---
	i := 1
	for ; i < len(recs); i++ {
		r := recs[i]
		d := r.data
		switch r.id {
		case recEOF:
			goto globalsDone
		case recFilePass:
			pw := password
			if pw == "" {
				pw = defaultXLSPassword // "write protected" only
			}
			key, err := filePassKey(d, pw)
			if errors.Is(err, ErrWrongPassword) && password == "" {
				return nil, ErrPasswordRequired
			}
			if err != nil {
				return nil, fmt.Errorf("xls: %w", err)
			}
			decryptBIFF(stream, recs, i, key)
		case recBoundSheet:
			if len(d) < 8 {
				continue
			}
			pos := int(binary.LittleEndian.Uint32(d))
			hs, dt := d[4]&3, d[5]
			if dt != 0 { // 0=worksheet, 1=macro, 2=chart, 6=VB
				continue
			}
			name, _ := readShortXLString(d[6:])
			sheets = append(sheets, xlsSheetRef{name: name, pos: pos, hidden: hs != 0})
		case recDateMode:
			date1904 = len(d) >= 2 && binary.LittleEndian.Uint16(d) == 1
		case recXF:
			if len(d) >= 4 {
				xfFmt = append(xfFmt, binary.LittleEndian.Uint16(d[2:]))
				xfRaw = append(xfRaw, d)
			}
		case recFont:
			fonts = append(fonts, d)
		case recFormat:
			if len(d) >= 5 {
				s, _ := readXLString(d[2:])
				formats[binary.LittleEndian.Uint16(d)] = s
			}
		case recSST:
			segs := [][]byte{d}
			for i+1 < len(recs) && recs[i+1].id == recContinue {
				i++
				segs = append(segs, recs[i].data)
			}
			sst = parseSST(segs)
		case recMsoDrawingGrp:
			dggData = append(dggData, d...)
			for i+1 < len(recs) && (recs[i+1].id == recContinue || recs[i+1].id == recMsoDrawingGrp) {
				i++
				dggData = append(dggData, recs[i].data...)
			}
		}
	}
globalsDone:

	fmtCache := map[uint16]fmtInfo{}
	xfInfo := func(ixfe uint16) fmtInfo {
		if int(ixfe) >= len(xfFmt) {
			return fmtInfo{}
		}
		id := xfFmt[ixfe]
		if fi, ok := fmtCache[id]; ok {
			return fi
		}
		fi := classifyFormat(int(id), formats[id])
		fmtCache[id] = fi
		return fi
	}

	blips := parseBlipStore(dggData)
	xfStyle := buildXFStyles(xfRaw, fonts)

	// index records by stream offset to locate sheet substreams
	byPos := make(map[int]int, len(recs))
	for k, r := range recs {
		byPos[r.pos] = k
	}

	wb := &Workbook{}
	for _, sref := range sheets {
		start, ok := byPos[sref.pos]
		if !ok || recs[start].id != recBOF {
			continue
		}
		sh := &Sheet{Name: sref.name, Hidden: sref.hidden}
		var drawing []byte
		pendR, pendC, pendXF := -1, -1, 0
		setXF := func(r, c, ixfe int, cell Cell) {
			if ixfe < len(xfStyle) {
				cell.Fmt = xfStyle[ixfe]
			}
			sh.set(r, c, cell)
		}
		depth := 0
	loop:
		for k := start; k < len(recs); k++ {
			r := recs[k]
			d := r.data
			switch r.id {
			case recBOF:
				depth++
				continue
			case recEOF:
				depth--
				if depth == 0 {
					break loop
				}
				continue
			}
			if depth != 1 { // embedded chart substream
				continue
			}
			switch r.id {
			case recLabelSST:
				if len(d) >= 10 {
					rw, col := u16(d, 0), u16(d, 2)
					idx := int(binary.LittleEndian.Uint32(d[6:]))
					if idx < len(sst) {
						setXF(rw, col, u16(d, 4), Cell{Kind: CellString, Str: sst[idx]})
					}
				}
			case recLabel, recRString:
				if len(d) >= 9 {
					s, _ := readXLString(d[6:])
					setXF(u16(d, 0), u16(d, 2), u16(d, 4), Cell{Kind: CellString, Str: s})
				}
			case recNumber:
				if len(d) >= 14 {
					v := math.Float64frombits(binary.LittleEndian.Uint64(d[6:]))
					setXF(u16(d, 0), u16(d, 2), u16(d, 4), numberCell(v, xfInfo(uint16(u16(d, 4))), date1904))
				}
			case recRK:
				if len(d) >= 10 {
					v := decodeRK(binary.LittleEndian.Uint32(d[6:]))
					setXF(u16(d, 0), u16(d, 2), u16(d, 4), numberCell(v, xfInfo(uint16(u16(d, 4))), date1904))
				}
			case recMulRK:
				if len(d) >= 6 {
					rw, col := u16(d, 0), u16(d, 2)
					for p := 4; p+6 <= len(d)-2; p += 6 {
						v := decodeRK(binary.LittleEndian.Uint32(d[p+2:]))
						setXF(rw, col, u16(d, p), numberCell(v, xfInfo(uint16(u16(d, p))), date1904))
						col++
					}
				}
			case recBoolErr:
				if len(d) >= 8 {
					rw, col := u16(d, 0), u16(d, 2)
					if d[7] == 0 {
						setXF(rw, col, u16(d, 4), Cell{Kind: CellBool, Bool: d[6] != 0})
					} else {
						setXF(rw, col, u16(d, 4), Cell{Kind: CellError, Str: biffErr(d[6])})
					}
				}
			case recFormula:
				if len(d) < 14 {
					continue
				}
				rw, col := u16(d, 0), u16(d, 2)
				if d[12] == 0xFF && d[13] == 0xFF {
					switch d[6] {
					case 0: // string result follows in STRING record
						pendR, pendC, pendXF = rw, col, u16(d, 4)
					case 1:
						setXF(rw, col, u16(d, 4), Cell{Kind: CellBool, Bool: d[8] != 0})
					case 2:
						setXF(rw, col, u16(d, 4), Cell{Kind: CellError, Str: biffErr(d[8])})
					}
				} else {
					v := math.Float64frombits(binary.LittleEndian.Uint64(d[6:]))
					setXF(rw, col, u16(d, 4), numberCell(v, xfInfo(uint16(u16(d, 4))), date1904))
				}
			case recString:
				if pendR >= 0 {
					s, _ := readXLString(d)
					setXF(pendR, pendC, pendXF, Cell{Kind: CellString, Str: s})
					pendR, pendC = -1, -1
				}
			case recMergeCells:
				if len(d) >= 2 {
					n := u16(d, 0)
					for j := 0; j < n && 2+j*8+8 <= len(d); j++ {
						p := 2 + j*8
						sh.Merges = append(sh.Merges, Range{R1: u16(d, p), R2: u16(d, p+2), C1: u16(d, p+4), C2: u16(d, p+6)})
					}
				}
			case recMsoDrawing:
				drawing = append(drawing, d...)
				for k+1 < len(recs) && recs[k+1].id == recContinue {
					k++
					drawing = append(drawing, recs[k].data...)
				}
			}
		}
		sh.Images = parseSheetDrawing(drawing, blips)
		wb.Sheets = append(wb.Sheets, sh)
	}
	return wb, nil
}

func u16(b []byte, off int) int { return int(binary.LittleEndian.Uint16(b[off:])) }

func decodeRK(rk uint32) float64 {
	var v float64
	if rk&2 != 0 {
		v = float64(int32(rk) >> 2)
	} else {
		v = math.Float64frombits(uint64(rk&0xFFFFFFFC) << 32)
	}
	if rk&1 != 0 {
		v /= 100
	}
	return v
}

func biffErr(code byte) string {
	switch code {
	case 0x00:
		return "#NULL!"
	case 0x07:
		return "#DIV/0!"
	case 0x0F:
		return "#VALUE!"
	case 0x17:
		return "#REF!"
	case 0x1D:
		return "#NAME?"
	case 0x24:
		return "#NUM!"
	case 0x2A:
		return "#N/A"
	}
	return "#ERR!"
}

func decodeChars(b []byte, high bool) string {
	if !high {
		r := make([]rune, len(b))
		for i, c := range b {
			r[i] = rune(c)
		}
		return string(r)
	}
	u := make([]uint16, len(b)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	return string(utf16.Decode(u))
}

// readShortXLString: cch(1) flags(1) chars
func readShortXLString(b []byte) (string, int) {
	if len(b) < 2 {
		return "", len(b)
	}
	n, high := int(b[0]), b[1]&1 != 0
	w := n
	if high {
		w *= 2
	}
	if 2+w > len(b) {
		w = len(b) - 2
	}
	return decodeChars(b[2:2+w], high), 2 + w
}

// readXLString: cch(2) flags(1) [rich/ext ignored] chars
func readXLString(b []byte) (string, int) {
	if len(b) < 3 {
		return "", len(b)
	}
	n, flags := u16(b, 0), b[2]
	p := 3
	if flags&0x08 != 0 {
		p += 2
	}
	if flags&0x04 != 0 {
		p += 4
	}
	w := n
	if flags&1 != 0 {
		w *= 2
	}
	if p+w > len(b) {
		w = len(b) - p
		if w < 0 {
			return "", len(b)
		}
	}
	return decodeChars(b[p:p+w], flags&1 != 0), p + w
}

// sstReader reads across SST + CONTINUE segments.
type sstReader struct {
	segs [][]byte
	si   int
	pos  int
}

func (r *sstReader) ensure() bool {
	for r.si < len(r.segs) && r.pos >= len(r.segs[r.si]) {
		r.si++
		r.pos = 0
	}
	return r.si < len(r.segs)
}

func (r *sstReader) bytes(n int) []byte {
	out := make([]byte, 0, n)
	for n > 0 && r.ensure() {
		seg := r.segs[r.si]
		k := min(len(seg)-r.pos, n)
		out = append(out, seg[r.pos:r.pos+k]...)
		r.pos += k
		n -= k
	}
	return out
}

func (r *sstReader) skip(n int) { r.bytes(n) }

func parseSST(segs [][]byte) []string {
	if len(segs) == 0 || len(segs[0]) < 8 {
		return nil
	}
	unique := int(binary.LittleEndian.Uint32(segs[0][4:]))
	r := &sstReader{segs: segs, pos: 8}
	out := make([]string, 0, unique)
	for len(out) < unique && r.ensure() {
		hdr := r.bytes(3)
		if len(hdr) < 3 {
			break
		}
		nch, flags := int(binary.LittleEndian.Uint16(hdr)), hdr[2]
		runs, ext := 0, 0
		if flags&0x08 != 0 {
			runs = int(binary.LittleEndian.Uint16(r.bytes(2)))
		}
		if flags&0x04 != 0 {
			ext = int(binary.LittleEndian.Uint32(r.bytes(4)))
		}
		high := flags&1 != 0
		var sb strings.Builder
		got := 0
		for got < nch {
			if r.si >= len(r.segs) {
				break
			}
			seg := r.segs[r.si]
			avail := len(seg) - r.pos
			if high {
				avail /= 2
			}
			if avail > nch-got {
				avail = nch - got
			}
			if avail > 0 {
				w := avail
				if high {
					w *= 2
				}
				sb.WriteString(decodeChars(seg[r.pos:r.pos+w], high))
				r.pos += w
				got += avail
			}
			if got < nch { // continue record: first byte = new flags
				r.si++
				r.pos = 0
				if r.si >= len(r.segs) || len(r.segs[r.si]) == 0 {
					break
				}
				high = r.segs[r.si][0]&1 != 0
				r.pos = 1
			}
		}
		r.skip(4*runs + ext)
		out = append(out, sb.String())
	}
	return out
}

// ---------------- Office Art (Escher) drawing layer ----------------

type escherRec struct {
	ver  uint16
	inst uint16
	typ  uint16
	body []byte
}

func escherRecords(b []byte) []escherRec {
	var out []escherRec
	for p := 0; p+8 <= len(b); {
		vi := binary.LittleEndian.Uint16(b[p:])
		typ := binary.LittleEndian.Uint16(b[p+2:])
		n := int(binary.LittleEndian.Uint32(b[p+4:]))
		end := p + 8 + n
		if n < 0 || end > len(b) {
			end = len(b)
		}
		out = append(out, escherRec{ver: vi & 0xF, inst: vi >> 4, typ: typ, body: b[p+8 : end]})
		p = end
	}
	return out
}

type blipEntry struct {
	data []byte
	ext  string
}

// parseBlipStore returns the BStore entries (1-based pib -> index pib-1).
func parseBlipStore(dgg []byte) []blipEntry {
	var out []blipEntry
	for _, top := range escherRecords(dgg) {
		if top.typ != 0xF000 {
			continue
		}
		for _, r := range escherRecords(top.body) {
			if r.typ != 0xF001 {
				continue
			}
			for _, fbse := range escherRecords(r.body) {
				var e blipEntry
				if fbse.typ == 0xF007 && len(fbse.body) >= 36 {
					cbName := int(fbse.body[33])
					off := 36 + cbName
					if off < len(fbse.body) {
						if bl := escherRecords(fbse.body[off:]); len(bl) > 0 {
							e = decodeBlip(bl[0])
						}
					}
				}
				out = append(out, e) // keep position even if empty
			}
		}
	}
	return out
}

func decodeBlip(r escherRec) blipEntry {
	b := r.body
	uid := func(singles ...uint16) int {
		if slices.Contains(singles, r.inst) {
			return 16
		}
		return 32 // rgbUid1 + rgbUid2
	}
	switch r.typ {
	case 0xF01D, 0xF02A: // JPEG (RGB / CMYK)
		return blipSlice(b, uid(0x46A, 0x6E2)+1, ".jpeg")
	case 0xF01E:
		return blipSlice(b, uid(0x6E0)+1, ".png")
	case 0xF029:
		return blipSlice(b, uid(0x6E4)+1, ".tiff")
	case 0xF01F: // DIB -> BMP
		e := blipSlice(b, uid(0x7A8)+1, ".bmp")
		e.data = dibToBMP(e.data)
		return e
	case 0xF01A, 0xF01B, 0xF01C: // EMF, WMF, PICT: 34-byte metafile header + (usually) deflated data
		off := uid(0x3D4, 0x216, 0x542)
		ext := map[uint16]string{0xF01A: ".emf", 0xF01B: ".wmf", 0xF01C: ".pict"}[r.typ]
		if off+34 > len(b) {
			return blipEntry{}
		}
		data := b[off+34:]
		if b[off+32] == 0 { // fCompression == msocompressionDeflate
			if zr, err := zlib.NewReader(bytes.NewReader(data)); err == nil {
				if dec, err := io.ReadAll(zr); err == nil {
					data = dec
				}
			}
		}
		return blipEntry{data: data, ext: ext}
	}
	return blipEntry{}
}

func blipSlice(b []byte, off int, ext string) blipEntry {
	if off >= len(b) {
		return blipEntry{}
	}
	return blipEntry{data: b[off:], ext: ext}
}

func dibToBMP(dib []byte) []byte {
	if len(dib) < 40 {
		return dib
	}
	hdrSize := int(binary.LittleEndian.Uint32(dib))
	bitCount := int(binary.LittleEndian.Uint16(dib[14:]))
	compression := binary.LittleEndian.Uint32(dib[16:])
	clrUsed := int(binary.LittleEndian.Uint32(dib[32:]))
	if clrUsed == 0 && bitCount <= 8 {
		clrUsed = 1 << bitCount
	}
	extra := 0
	if compression == 3 && hdrSize == 40 { // BI_BITFIELDS
		extra = 12
	}
	off := 14 + hdrSize + extra + clrUsed*4
	h := make([]byte, 14, 14+len(dib))
	h[0], h[1] = 'B', 'M'
	binary.LittleEndian.PutUint32(h[2:], uint32(14+len(dib)))
	binary.LittleEndian.PutUint32(h[10:], uint32(off))
	return append(h, dib...)
}

// parseSheetDrawing walks the sheet's OfficeArtDgContainer and returns
// picture shapes with their top-left anchor cell.
func parseSheetDrawing(dg []byte, blips []blipEntry) []Image {
	if len(dg) == 0 || len(blips) == 0 {
		return nil
	}
	var out []Image
	type anchor struct{ row, col int }
	var walkSpgr func(body []byte, parent *anchor)
	walkSp := func(body []byte, parent *anchor) (*anchor, int) {
		var a *anchor
		pib := 0
		for _, r := range escherRecords(body) {
			switch r.typ {
			case 0xF010: // ClientAnchor (sheet)
				if len(r.body) >= 18 {
					a = &anchor{row: u16(r.body, 6), col: u16(r.body, 2)}
				}
			case 0xF00B, 0xF122: // FOPT / tertiary FOPT
				n := int(r.inst)
				for j := 0; j < n && j*6+6 <= len(r.body); j++ {
					opid := binary.LittleEndian.Uint16(r.body[j*6:])
					val := int(binary.LittleEndian.Uint32(r.body[j*6+2:]))
					if opid&0x3FFF == 0x0104 && opid&0x8000 == 0 { // pib
						pib = val
					}
				}
			}
		}
		if a == nil {
			a = parent
		}
		return a, pib
	}
	walkSpgr = func(body []byte, parent *anchor) {
		groupAnchor := parent
		for idx, r := range escherRecords(body) {
			switch r.typ {
			case 0xF004:
				a, pib := walkSp(r.body, groupAnchor)
				if idx == 0 { // first shape of a group = group shape itself
					if a != nil {
						groupAnchor = a
					}
					continue
				}
				if pib > 0 && pib <= len(blips) && a != nil && len(blips[pib-1].data) > 0 {
					bl := blips[pib-1]
					out = append(out, Image{Row: a.row, Col: a.col, Data: bl.data, Ext: bl.ext})
				}
			case 0xF003: // nested group
				walkSpgr(r.body, groupAnchor)
			}
		}
	}
	for _, top := range escherRecords(dg) {
		if top.typ != 0xF002 {
			continue
		}
		for _, r := range escherRecords(top.body) {
			if r.typ == 0xF003 {
				walkSpgr(r.body, nil)
			}
		}
	}
	return out
}

// buildXFStyles turns XF + FONT records into shared CellFormat values.
func buildXFStyles(xfs, fonts [][]byte) []*CellFormat {
	cache := map[string]*CellFormat{}
	out := make([]*CellFormat, len(xfs))
	for i, x := range xfs {
		cf := CellFormat{}
		if len(x) >= 20 {
			fls := binary.LittleEndian.Uint32(x[14:]) >> 26
			icv := binary.LittleEndian.Uint16(x[18:])
			if fls != 0 {
				cf.Fill = fmt.Sprintf("pal%d/%d", icv&0x7F, fls)
			}
			ifnt := u16(x, 0)
			if ifnt >= 4 { // font index 4 does not exist in BIFF
				ifnt--
			}
			if ifnt < len(fonts) && len(fonts[ifnt]) >= 8 {
				f := fonts[ifnt]
				cf.FontSize = float64(u16(f, 0)) / 20
				cf.Italic = f[2]&0x02 != 0
				cf.FontColor = fmt.Sprintf("pal%d", u16(f, 4))
				cf.Bold = u16(f, 6) >= 700
			}
		}
		k := cf.Key()
		if c, ok := cache[k]; ok {
			out[i] = c
			continue
		}
		p := cf
		cache[k] = &p
		out[i] = &p
	}
	return out
}

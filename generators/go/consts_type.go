package _go

const (
	formatType = `// %sPsqlType interface for reading data from psql connection
type (
	%[1]sPsqlType struct {
		%[1]sFields
		Status      pgtype.Status 
		convertErrs []string
	}
)

// GetFields implement dbEngine.RowScanner interface
func (r *%[1]sPsqlType) GetFields(columns []dbEngine.Column) []any {
	v := make([]any, len(columns))
	for i, col := range columns {
		switch name:= col.Name(); name {
		case "themes":
			v[i] = r
		default:
			v[i] = r.RefColValue(name)
		}
	}

	return v
}
// Set implement pgtype.Value interface
func (dst *%[1]sPsqlType) Set(src any) error {
	switch value := src.(type) {
	// untyped nil and typed nil interfaces are different
	case nil:
		*dst = %[1]sPsqlType{Status: pgtype.Null}
		return nil
	case %[1]sPsqlType:
		*dst = %[1]sPsqlType{
			%[1]sFields: value.%[1]sFields,
			Status:          pgtype.Present,
		}
		return nil

	default:
		return nil
	}
}
// Get implement pgtype.Value interface
func (dst *%[1]sPsqlType) Get() any {
	switch dst.Status {
	case pgtype.Present:
		return *dst
	case pgtype.Null:
		return nil
	default:
		return dst.Status
	}
}
// AssignTo implement pgtype.Value interface
func (src *%[1]sPsqlType) AssignTo(dst any) error {
	switch src.Status {
	case pgtype.Present:
		switch v := dst.(type) {
		case *%[1]sPsqlType:
			(*v).%[1]sFields = src.%[1]sFields
			return nil

		default:
			if nextDst, retry := pgtype.GetAssignToDstType(dst); retry {
				return src.AssignTo(nextDst)
			}
		}
		return nil

	case pgtype.Null:
		return pgtype.NullAssignTo(dst)

	default:
		return errors.Errorf("cannot decode %%v into %%T", src, dst)
	}
}
// DecodeText implement pgtype.TextDecoder interface
func (dst *%[1]sPsqlType) DecodeText(ci *pgtype.ConnInfo, src []byte) error {
	if len(src) == 0 {
		*dst = %[1]sPsqlType{Status: pgtype.Null}
		return nil
	}

	*dst = dst.readSrc(ci, src)

	return nil
}
// DecodeBinary implement pgtype.DecodeBinary interface
func (dst *%[1]sPsqlType) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	if len(src) == 0 {
		*dst = %[1]sPsqlType{Status: pgtype.Null}
		return nil
	}

	*dst = dst.readSrc(ci, src)

	return nil
}
func (dst *%[1]sPsqlType) readSrc(ci *pgtype.ConnInfo, src []byte) %[1]sPsqlType {
	srcPart := bytes.Split(src[1:len(src)-1], []byte(","))
	return %[1]sPsqlType{
		%[1]sFields: %[1]sFields{
			// fill columns of table
			%s
		},
		Status: pgtype.Present,
	}
}`
)

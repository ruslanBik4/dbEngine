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

func (dst *%[1]sPsqlType) Set(src interface{}) error {
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

func (dst *%[1]sPsqlType) Get() interface{} {
	switch dst.Status {
	case pgtype.Present:
		return *dst
	case pgtype.Null:
		return nil
	default:
		return dst.Status
	}
}

func (src *%[1]sPsqlType) AssignTo(dst interface{}) error {
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

func (dst *%[1]sPsqlType) DecodeText(ci *pgtype.ConnInfo, src []byte) error {
	if len(src) == 0 {
		*dst = %[1]sPsqlType{Status: pgtype.Null}
		return nil
	}

	srcPart := bytes.Split(src[1:len(src)-1], []byte(","))

	record := %[1]sFields{
	// fill columns of table
`
	formatEnd = `
	}

	*dst = %[1]sPsqlType{%[1]sFields: record, Status: pgtype.Present}

	return nil
}`
)

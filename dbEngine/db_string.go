package dbEngine

import (
	"go/types"
)

type StringColumn struct {
	comment, name            string
	req, primary, isNullable bool
}

func (s *StringColumn) AutoIncrement() bool {
	return false
}

func (c *StringColumn) IsNullable() bool {
	return c.isNullable
}

func (s *StringColumn) Default() string {
	return ""
}

func NewStringColumn(name, comment string, req bool) *StringColumn {
	return &StringColumn{comment, name, req, false, false}
}

func (s *StringColumn) BasicType() types.BasicKind {
	return types.String
}

func (s *StringColumn) BasicTypeInfo() types.BasicInfo {
	return types.IsString
}

func (s *StringColumn) CheckAttr(fieldDefine string) string {
	return ""
}

func (s *StringColumn) Comment() string {
	return s.comment
}

func (s *StringColumn) Primary() bool {
	return s.primary
}

func (s *StringColumn) Type() string {
	return "string"
}

func (s *StringColumn) Required() bool {
	return s.req
}

func (s *StringColumn) Name() string {
	return s.name
}

func (s *StringColumn) CharacterMaximumLength() int {
	return len(s.name)
}

func (c *StringColumn) SetNullable(f bool) {
	c.isNullable = f
}

func SimpleColumns(names ...string) []Column {
	s := make([]Column, len(names))
	for i, name := range names {
		s[i] = NewStringColumn(name, name, false)
	}

	return s
}

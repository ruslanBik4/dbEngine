package dbEngine

import (
	"go/types"
)

type StringColumn struct {
	comment, name, colDefault string
	req, primary, isNullable  bool
	maxLen                    int
}

func (s *StringColumn) AutoIncrement() bool {
	return false
}

func (c *StringColumn) IsNullable() bool {
	return c.isNullable
}

func (s *StringColumn) Default() interface{} {
	return ""
}

func (s *StringColumn) SetDefault(str interface{}) {
	s.colDefault = str.(string)
}

func NewStringColumn(name, comment string, req bool, maxLen ...int) *StringColumn {
	if len(maxLen) == 0 {
		maxLen = append(maxLen, 0)
	}

	return &StringColumn{
		comment: comment,
		name:    name,
		req:     req,
		maxLen:  maxLen[0],
	}
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
	return s.maxLen
}

func (c *StringColumn) SetNullable(f bool) {
	c.isNullable = f
}

func (c *StringColumn) Foreign() *ForeignKey {
	return nil
}

func SimpleColumns(names ...string) []Column {
	s := make([]Column, len(names))
	for i, name := range names {
		s[i] = NewStringColumn(name, name, false)
	}

	return s
}

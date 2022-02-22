package dbEngine

import (
	"go/types"
)

type StringColumn struct {
	comment, name, colDefault string
	req, primary, isNullable  bool
	maxLen                    int
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

func (s *StringColumn) CharacterMaximumLength() int {
	return s.maxLen
}

func (s *StringColumn) Comment() string {
	return s.comment
}

func (s *StringColumn) Name() string {
	return s.name
}

func (s *StringColumn) AutoIncrement() bool {
	return false
}

func (s *StringColumn) IsNullable() bool {
	return s.isNullable
}

func (s *StringColumn) Default() interface{} {
	return ""
}

func (s *StringColumn) SetDefault(str interface{}) {
	s.colDefault = str.(string)
}

func (s *StringColumn) Foreign() *ForeignKey {
	return nil
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

func (s *StringColumn) SetNullable(f bool) {
	s.isNullable = f
}

func SimpleColumns(names ...string) []Column {
	s := make([]Column, len(names))
	for i, name := range names {
		s[i] = NewStringColumn(name, name, false)
	}

	return s
}

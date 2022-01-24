package typesExt

import (
	"go/types"
)

const (
	TArray  types.BasicKind = -1
	TMap    types.BasicKind = -2
	TStruct types.BasicKind = -3
)

var stringExtTypes = map[types.BasicKind]string{
	TArray:  "array",
	TMap:    "map",
	TStruct: "struct",
}

// StringTypeKinds return describe of type
func StringTypeKinds(typ types.BasicKind) string {
	if typ < 0 {
		return stringExtTypes[typ]
	}
	return types.Typ[typ].String()
}

// Basic type of typ
func Basic(typ types.BasicKind) *types.Basic {
	if typ < 0 {
		typ = types.UnsafePointer
	}

	return types.Typ[typ]
}

// BasicInfo abount type
func BasicInfo(typ types.BasicKind) types.BasicInfo {
	if typ < 0 {
		typ = types.UnsafePointer
	}

	return types.Typ[typ].Info()
}

// IsNumeric returns true for numeric types
func IsNumeric(b types.BasicInfo) bool {

	return (b & types.IsNumeric) != 0
}

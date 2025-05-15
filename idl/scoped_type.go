package idl

// ScopedType 表示带有作用域的类型，例如 A::B::C
type ScopedType struct {
	BaseType
	Name string // 包含完整作用域的类型名称，例如 "module::submodule::type"
}

// TypeName 返回类型全名（包括作用域）
func (t *ScopedType) TypeName() string {
	return t.Name
}

// GoTypeName 返回对应的Go类型名
func (t *ScopedType) GoTypeName() string {
	// 将CORBA作用域操作符 :: 转换为Go的包路径分隔符 .
	return t.Name // 在代码生成时进一步处理
}

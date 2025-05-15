package main

import (
	"fmt"
	"os"

	"github.com/ifabos/go-corba/idl"
)

func main() {
	parser := idl.NewParser()

	// 设置当前文件名，用于错误报告
	filePath := "examples/idl/complex_test.idl"
	parser.SetCurrentFile(filePath)

	// 打开测试文件
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("打开文件出错: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// 解析IDL文件
	err = parser.Parse(file)
	if err != nil {
		fmt.Printf("解析错误: %v\n", err)
		return // 使用return而不是os.Exit以允许defer执行
	}

	// 获取解析结果
	rootModule := parser.GetRootModule()
	fmt.Println("解析成功！")

	// 递归打印模块信息
	printModule(rootModule, 0)
}

func printModule(module *idl.Module, depth int) {
	// 创建缩进
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}

	// 打印模块名
	if module.Name == "" {
		fmt.Printf("%s[根模块]\n", indent)
	} else {
		fmt.Printf("%s[模块: %s]\n", indent, module.Name)
	}

	// 打印类型
	for name, typ := range module.Types {
		fmt.Printf("%s  类型: %s (%T)\n", indent, name, typ)

		// 如果是结构体，打印字段
		if structType, ok := typ.(*idl.StructType); ok {
			fmt.Printf("%s    字段:\n", indent)
			for _, field := range structType.Fields {
				fmt.Printf("%s      - %s: %s\n", indent, field.Name, field.Type.TypeName())
			}
		}
	}

	// 递归打印子模块
	for _, submodule := range module.Submodules {
		printModule(submodule, depth+1)
	}
}

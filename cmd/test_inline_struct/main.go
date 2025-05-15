package main

import (
	"fmt"
	"os"

	"github.com/ifabos/go-corba/idl"
)

func main() {
	parser := idl.NewParser()

	// 设置当前文件名，用于错误报告
	filePath := "examples/idl/inlinstruct.idl"
	parser.SetCurrentFile(filePath)

	// 打开测试文件
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// 解析IDL文件
	err = parser.Parse(file)
	if err != nil {
		fmt.Printf("Error parsing IDL file: %v\n", err)
		os.Exit(1)
	}

	// 获取解析结果
	rootModule := parser.GetRootModule()

	// 打印解析结果
	fmt.Println("解析成功！")

	// 检查是否解析了test模块
	testModule, exists := rootModule.GetSubmodule("test")
	if !exists {
		fmt.Println("未找到test模块")
		os.Exit(1)
	}

	// 检查是否解析了Address类型
	addressType, exists := testModule.Types["Address"]
	if !exists {
		fmt.Println("未找到Address类型")
		os.Exit(1)
	}

	// 打印Address类型信息
	fmt.Printf("找到Address类型: %s\n", addressType.TypeName())

	// 如果是StructType，打印其字段
	if structType, ok := addressType.(*idl.StructType); ok {
		fmt.Println("Address是结构体类型，包含以下字段:")
		for _, field := range structType.Fields {
			fmt.Printf("  - %s: %s\n", field.Name, field.Type.TypeName())
		}
	}
}

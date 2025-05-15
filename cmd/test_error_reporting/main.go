package main

import (
	"fmt"
	"os"

	"github.com/ifabos/go-corba/idl"
)

func main() {
	parser := idl.NewParser()

	// 设置当前文件名，用于错误报告
	filePath := "examples/idl/error_struct.idl"
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
		os.Exit(1)
	}

	// 如果没有错误，这是意外的
	fmt.Println("解析成功（应该失败的）！")
}

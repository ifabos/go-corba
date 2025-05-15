package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ifabos/go-corba/idl"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("用法: go run main.go <idl文件路径>")
		os.Exit(1)
	}

	filePath := os.Args[1]
	absPath, _ := filepath.Abs(filePath)

	parser := idl.NewParser()
	parser.SetCurrentFile(absPath)

	// 打开IDL文件
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

		// 打印文件内容，帮助定位问题
		file.Seek(0, 0) // 重置文件指针到开始位置
		content, _ := os.ReadFile(filePath)
		fmt.Println("文件内容:")
		fmt.Println(string(content))

		os.Exit(1)
	}

	fmt.Println("解析成功！")
}

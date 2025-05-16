package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ifabos/go-corba/idl"
)

func main() {
	// 获取当前工作目录
	workDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("获取工作目录失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("当前工作目录: %s\n", workDir)

	// 创建IDL解析器
	parser := idl.NewParser()

	// 设置include处理器
	parser.SetIncludeHandler(func(path string) (io.Reader, error) {
		fmt.Printf("尝试加载include文件: %s\n", path)
		file, err := os.Open(path)
		if err != nil {
			fmt.Printf("加载include文件失败: %v\n", err)
		}
		return file, err
	})

	// 添加示例IDL目录到include路径
	examplesDir := filepath.Join(workDir, "examples", "idl")
	parser.AddIncludeDir(examplesDir)
	fmt.Printf("添加include路径: %s\n", examplesDir)

	// 测试文件路径
	testFile := filepath.Join(examplesDir, "struct_typedef_test.idl")
	parser.SetCurrentFile(testFile)

	fmt.Printf("解析文件: %s\n", testFile)

	// 检查文件是否存在
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		fmt.Printf("文件不存在: %s\n", testFile)
		os.Exit(1)
	}

	// 打开测试文件
	file, err := os.Open(testFile)
	if err != nil {
		fmt.Printf("打开文件错误: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// 打印文件内容
	fileContent, err := io.ReadAll(file)
	if err != nil {
		fmt.Printf("读取文件内容错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("文件内容:")
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println(string(fileContent))
	fmt.Println(strings.Repeat("-", 40))

	// 需要重新打开文件因为已经读到文件末尾
	file.Close()
	file, err = os.Open(testFile)
	if err != nil {
		fmt.Printf("重新打开文件错误: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// 解析IDL文件
	err = parser.Parse(file)
	if err != nil {
		fmt.Printf("解析错误: %v\n", err)
		os.Exit(1)
	}

	// 获取解析结果
	rootModule := parser.GetRootModule()

	// 打印所有定义的类型
	fmt.Println("解析成功！定义的类型:")
	for name, typ := range rootModule.AllTypes() {
		fmt.Printf("  %s: %s\n", name, typ.TypeName())
	}
}
